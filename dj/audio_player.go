package dj

import (
	"encoding/binary"
	"fmt"
	"github.com/Ayvan/ninjam-dj-bot/tracks"
	"github.com/azul3d/engine/audio"
	"github.com/burillo-se/ninjamencoder"
	"github.com/hajimehoshi/go-mp3"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"os"
	"path"
	"runtime/debug"
	"time"
)

// channels всегда 2 т.к. используемый MP3-декодер всегда отдаёт звук в стерео
const channels = 2

type IntervalBeginWriter interface {
	IntervalBegin(guid [16]byte, channelIndex uint8)
	IntervalWrite(guid [16]byte, data []byte, flags uint8)
}

type JamPlayer struct {
	track      *tracks.Track
	tracksPath string
	source     io.Reader
	sampleRate int
	bpm        uint
	bpi        uint
	repeats    int
	ninjamBot  IntervalBeginWriter
	stop       chan bool
	playing    bool
}

type AudioInterval struct {
	GUID         [16]byte
	ChannelIndex uint8
	Flags        uint8
	Data         [][]byte
	index        int // index of current audio data block
}

func NewJamPlayer(tracksPath string, ninjamBot IntervalBeginWriter) *JamPlayer {
	return &JamPlayer{ninjamBot: ninjamBot, tracksPath: tracksPath, stop: make(chan bool, 1)}
}

func (jp *JamPlayer) Playing() bool {
	return jp.playing
}

func (jp *JamPlayer) Track() *tracks.Track {
	return jp.track
}

func (jp *JamPlayer) LoadTrack(track *tracks.Track) {
	jp.track = track
	filePath := track.FilePath

	if !path.IsAbs(filePath) {
		filePath = path.Join(jp.tracksPath, filePath)
	}

	err := jp.setMP3Source(filePath)
	if err != nil {
		logrus.Error(err)
	}

	jp.setBPM(track.BPM)
	jp.setBPI(track.BPI)
	jp.SetRepeats(0)
}

func (jp *JamPlayer) SetRepeats(repeats int) {
	jp.repeats = repeats
}

func (jp *JamPlayer) setMP3Source(source string) error {
	jp.Stop() // stop before set new source

	out, err := os.OpenFile(source, os.O_RDONLY, 0664)
	if err != nil {
		return fmt.Errorf("setMP3Source error: %s", err)
	}

	decoder, err := mp3.NewDecoder(out)
	if err != nil {
		return fmt.Errorf("NewDecoder error: %s", err)
	}

	jp.source = decoder

	jp.sampleRate = decoder.SampleRate()

	return nil
}

func (jp *JamPlayer) setBPM(bpm uint) {
	jp.bpm = bpm
}

func (jp *JamPlayer) setBPI(bpi uint) {
	jp.bpi = bpi
}

func (jp *JamPlayer) Start() error {
	if jp.source == nil {
		fmt.Println("no source detected")
		return fmt.Errorf("no source detected")
	}

	jp.stop = make(chan bool, 1)

	// посчитаем на каких сэмплах у нас начало, и на каких конец зацикливания
	startTime := time.Duration(jp.track.LoopStart) * time.Microsecond
	loopStartPos := timeToSamples(startTime, jp.sampleRate)

	endTime := time.Duration(jp.track.LoopEnd) * time.Microsecond
	loopEndPos := timeToSamples(endTime, jp.sampleRate)

	intervalTime := (float64(time.Minute) / float64(jp.bpm)) * float64(jp.bpi)
	intervalSamples := int(math.Ceil(float64(jp.sampleRate) * intervalTime / float64(time.Second)))
	intervalSamplesChannels := intervalSamples * channels

	jp.playing = true

	samplesBuffer := make([][]float32, 2)

	// эта переменная будет установлена когда буфер будет заполнен всеми данными из MP3 файла
	bufferFull := false

	waitData := make(chan bool, 1)
	// это фоновая загрузка и декодирование MP3 в буфер
	go func() {
		intervalsReady := 0

		for {
			buf := audio.Float32{}.Make(intervalSamplesChannels, intervalSamplesChannels)
			rs, err := toReadSeeker(jp.source, intervalSamplesChannels)
			if err != nil && err != io.EOF && err.Error() != "end of stream" {
				logrus.Errorf("source.Read error: %s", err)
			}

			var n int
			n, err = rs.Read(buf)
			if err != nil && err != io.EOF && err.Error() != "end of stream" {
				logrus.Errorf("source.Read error: %s", err)
			}
			if n == 0 {
				bufferFull = true
				return
			}

			deinterleavedSamples, err := ninjamencoder.DeinterleaveSamples(buf.(audio.Float32), channels)
			if err != nil {
				logrus.Errorf("DeinterleaveSamples error: %s", err)
				return
			}

			for i := 0; i < channels; i++ {
				samplesBuffer[i] = append(samplesBuffer[i], deinterleavedSamples[i]...)
			}

			intervalsReady++
			if intervalsReady == 3 {
				waitData <- true
			}
		}
	}()

	// ждём пока будут готовы интервалы
	<-waitData

	// TODO на выходе функции ловить ошибку и сообщать в чат что трек прерван из-за ошибки
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("panic in JamPlayer.Start: %s", r)
				logrus.Error(string(debug.Stack()))
			}

			jp.playing = false
		}()

		ticker := time.NewTicker(time.Duration(intervalTime))

		oggEncoder := ninjamencoder.NewEncoder()
		oggEncoder.SampleRate = jp.sampleRate

		play := true
		currentPos := 0

		for play {
			deinterleavedSamples := make([][]float32, 2)
			endPos := currentPos + intervalSamples

			if endPos > len(samplesBuffer[0]) {
				endPos = len(samplesBuffer[0])
				play = false // дошли до конца - завершаем
			}

			if currentPos >= loopStartPos && endPos >= loopEndPos && jp.repeats > 0 {
				play = true // если ранее получили флаг остановки - значит снимем его, мы ушли в очередной цикл
				samplesToIntervalEnd := endPos - loopEndPos
				endPos = loopEndPos

				for i := 0; i < channels; i++ {
					// создаём новый слайс и копируем в него, т.к. дальше нам нужно с ним работать отдельно от кэшированного слайса - мы будем его менять через append
					deinterleavedSamples[i] = make([]float32, endPos-currentPos, intervalSamples)
					copy(deinterleavedSamples[i], samplesBuffer[i][currentPos:endPos])
					deinterleavedSamples[i] = append(deinterleavedSamples[i], samplesBuffer[i][loopStartPos:loopStartPos+samplesToIntervalEnd]...)
				}

				currentPos = loopStartPos + samplesToIntervalEnd

				jp.repeats--
				logrus.Debugf("repeats left: %d", jp.repeats)
			} else {
				for i := 0; i < channels; i++ {
					deinterleavedSamples[i] = samplesBuffer[i][currentPos:endPos]
				}

				currentPos = endPos
			}

			data, err := oggEncoder.EncodeNinjamInterval(deinterleavedSamples)
			if err != nil {
				logrus.Errorf("EncodeNinjamInterval error: %s", err)
				return
			}

			guid, _ := uuid.NewV1()

			select {
			case <-ticker.C:
			case <-jp.stop:
				ticker.Stop()
				return
			}

			interval := AudioInterval{
				GUID:         guid,
				ChannelIndex: 0,
				Flags:        0,
				Data:         data,
			}

			jp.ninjamBot.IntervalBegin(interval.GUID, interval.ChannelIndex)

			hasNext := true
			for hasNext {
				var intervalData []byte

				intervalData, hasNext = interval.next()

				if !hasNext {
					interval.Flags = 1
				}

				jp.ninjamBot.IntervalWrite(interval.GUID, intervalData, interval.Flags)
			}
		}
	}()

	return nil
}

func (jp *JamPlayer) Stop() {
	if jp.stop != nil && len(jp.stop) == 0 {
		jp.stop <- true
	}
}

func (ai *AudioInterval) next() (data []byte, hasNext bool) {
	hasNext = true
	if len(ai.Data) > ai.index {
		data = ai.Data[ai.index]
		ai.index++
	}
	if len(ai.Data) < ai.index+1 {
		hasNext = false
	}

	return
}

func toReadSeeker(reader io.Reader, samples int) (res audio.ReadSeeker, err error) {
	buf := audio.NewBuffer(audio.Float32{})
	res = buf

	for ; samples > 0; samples-- {
		data := make([]byte, 2, 2)
		var n int
		n, err = reader.Read(data)
		if err != nil && err != io.EOF {
			return
		}
		if n == 0 {
			err = nil // remove EOF error
			return
		}

		intData := int16(binary.LittleEndian.Uint16(data))
		buf.Write(audio.Float32{Int16ToFloat32(intData)})
	}

	return
}

func Int16ToFloat32(s int16) float32 {
	return float32(s) / float32(math.MaxInt16+1)
}

func timeToSamples(t time.Duration, sampleRate int) int {
	return int(math.Ceil(float64(sampleRate) * float64(t) / float64(time.Second)))

}
