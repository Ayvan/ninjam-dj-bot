package dj

import (
	"github.com/ayvan/ninjam-chatbot/models"
	"github.com/ayvan/ninjam-dj-bot/lib"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strings"
	"time"
)

const (
	messageAfter15Seconds = "%s's turn in 15 seconds"
	messageNowPlaying     = "%s is playing now"
	messageIsNext         = "%s is next"
)

func init() {
	message.SetString(language.Russian, messageAfter15Seconds, "очередь %s через 15 секунд")
	message.SetString(language.Russian, messageNowPlaying, "сейчас играет %s")
	message.SetString(language.Russian, messageIsNext, "готовится играть %s")
}

type QueueManager struct {
	botName           string
	userStartTime     *time.Time
	userPlayDuration  time.Duration
	trackEndTime      time.Time
	sendMessage       func(msg string)
	first             *user
	current           *user
	after15SecMsgSent bool // флаг что сообщение messageAfter15Seconds уже отправлено

	stopped     bool
	stopChannel chan bool
}

type user struct {
	Name string
	Prev *user
	Next *user
}

func NewQueueManager(botName string, sendMessageFunc func(msg string)) *QueueManager {
	qm := &QueueManager{botName: botName, sendMessage: sendMessageFunc}
	qm.stopChannel = make(chan bool, 1)
	go qm.supervisor()

	return qm
}

func (qm *QueueManager) Close() {
	qm.stopChannel <- true
	close(qm.stopChannel)
}

func (qm *QueueManager) supervisor() {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			if qm.stopped {
				continue
			}
			if qm.userStartTime == nil {
				continue
			}

			if qm.userStartTime.Add(qm.userPlayDuration).Before(time.Now()) &&
				qm.userStartTime.Add(qm.userPlayDuration+time.Second*15).After(time.Now()) {
				if qm.current != nil && qm.current.Next != nil && qm.sendMessage != nil && !qm.after15SecMsgSent {
					qm.sendMessage(p.Sprintf(messageAfter15Seconds, qm.current.Next.Name))
					qm.after15SecMsgSent = true
				}
				continue
			}
			if qm.userStartTime.Add(qm.userPlayDuration).After(time.Now()) {
				continue
			}
			// если до конца трека осталось менее чем qm.userPlayDuration то ничего не делаем
			if qm.trackEndTime.After(time.Now()) && qm.trackEndTime.Sub(time.Now()) < time.Second*15 {
				continue
			}
			qm.next()
		case <-qm.stopChannel:
			ticker.Stop()
			return
		}
	}
}

func (qm *QueueManager) UsersCount() (i uint) {
	users := qm.Users()
	return uint(len(users))
}

func (qm *QueueManager) Users() (users []string) {
	if qm.current == nil {
		return
	}

	i := 0
	curr := qm.current
	for {
		users = append(users, curr.Name)
		i++
		if curr.Next == nil {
			return
		}
		curr = curr.Next

		// shit happened...
		if i > 1000 {
			i = 0
			return
		}
	}

	return
}

func (qm *QueueManager) Add(userName string) {
	userName = cleanName(userName)
	if userName == qm.botName {
		return
	}

	qm.Del(userName)
	newUser := &user{Name: userName}
	if qm.current == nil {
		qm.current = newUser
		qm.first = newUser
		return
	}

	curr := qm.current
	for {
		if curr.Next == nil {
			curr.Next = newUser
			curr.Next.Prev = curr
			return
		}
		curr = curr.Next
	}
}

func (qm *QueueManager) Del(userName string) {
	userName = cleanName(userName)
	if userName == qm.botName {
		return
	}
	if qm.current == nil {
		return
	}
	curr := qm.current
	i := 0
	for {
		if curr == nil {
			return
		}
		if curr.Name == userName {
			if curr.Prev != nil {
				curr.Prev.Next = curr.Next
			}
			if curr.Next == nil && i == 0 {
				qm.current = nil
				return
			}

			if curr.Next != nil {
				curr.Next.Prev = curr.Prev
			}

			if i == 0 {
				qm.current = curr.Next
				qm.current.Prev = nil
				// если текущий юзер и есть выбывший - сразу переключаем
				qm.start(0)
			}

			return
		}
		curr = curr.Next
		i++
	}
}

func (qm *QueueManager) next() {
	if qm.current != nil && qm.current.Next != nil {
		next := qm.current.Next

		curr := qm.current
		// перекинем текущего в конец списка
		for {
			if curr.Next != nil {
				curr = curr.Next
				continue
			}
			curr.Next = qm.current
			curr.Next.Prev = curr
			curr.Next.Next = nil
			break
		}
		qm.current = next

		qm.start(0)
		return
	}

	// если следующего нет - просто обновим таймер и текущий продолжит играть
	tn := time.Now()
	qm.userStartTime = &tn
}

func (qm *QueueManager) start(intervalDuration time.Duration) {
	tn := time.Now().Add(intervalDuration)
	qm.userStartTime = &tn
	qm.after15SecMsgSent = false
	if qm.current != nil && qm.sendMessage != nil {
		if qm.current.Next == nil {
			qm.sendMessage(p.Sprintf(messageNowPlaying, qm.current.Name))
		} else {
			// TODO если до конца трека мало времени на ещё одного - не объявлять!
			qm.sendMessage(p.Sprintf(messageNowPlaying, qm.current.Name) + ", " + p.Sprintf(messageIsNext, qm.current.Next.Name))
		}
	}
}

func (qm *QueueManager) OnStart(trackDuration, intervalDuration time.Duration) {
	qm.userPlayDuration = lib.CalcUserPlayDuration(trackDuration)
	qm.trackEndTime = time.Now().Add(trackDuration)
	qm.start(intervalDuration)
}

func (qm *QueueManager) OnStop() {
	qm.stopped = true
}

func (qm *QueueManager) OnUserinfoChange(user models.UserInfo) {
	if user.Active == 0x1 {
		qm.Add(string(user.Name))
		return
	}
	qm.Del(string(user.Name))
}

func cleanName(userName string) string {
	i := strings.Index(userName, "@")
	if i < 0 {
		return userName
	}

	return userName[:i]
}
