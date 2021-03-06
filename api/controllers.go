package api

import (
	"github.com/ayvan/ninjam-dj-bot/auth"
	"github.com/ayvan/ninjam-dj-bot/config"
	"github.com/ayvan/ninjam-dj-bot/dj"
	"github.com/ayvan/ninjam-dj-bot/helpers"
	"github.com/ayvan/ninjam-dj-bot/tracks"
	"github.com/ayvan/ninjam-dj-bot/tracks_sync"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
)

type ErrorResp struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

var jamDB *tracks.JamDB
var authenticator *auth.JWTAuth

func Init(db *tracks.JamDB, aDB *auth.DB) {
	jamDB = db

	conf := auth.Config{
		PrivateKeyPath:       config.Get().PrivateKeyPath,
		PublicKeyPath:        config.Get().PublicKeyPath,
		DefaultAdminPassword: config.Get().DefaultAdminPassword,
	}
	var err error
	authenticator, err = auth.NewAuth(conf, aDB)
	if err != nil {
		logrus.Fatal(err)
	}
}

type LoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login POST /login
func Login(ctx echo.Context) error {
	req := LoginForm{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	token, err := authenticator.Authenticate(req.Username, req.Password)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, newError(http.StatusUnauthorized))
	}

	return ctx.JSON(http.StatusOK, token)
}

// Tracks GET /tracks
func Tracks(ctx echo.Context) error {
	t, err := jamDB.Tracks()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, t)
}

// Track GET /tracks/:id
func Track(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	t, err := jamDB.Track(uint(id))
	if err != nil {
		switch err {
		case tracks.ErrorNotFound:
			return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound, err.Error()))
		default:
			return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
		}
	}

	return ctx.JSON(http.StatusOK, t)
}

// Tags GET /tags
func Tags(ctx echo.Context) error {
	t, err := jamDB.Tags()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, t)
}

// Tag GET /tags/:id
func Tag(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		switch err {
		case tracks.ErrorNotFound:
			return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound, err.Error()))
		default:
			return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
		}
	}

	t, err := jamDB.Tag(uint(id))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, t)
}

// PutTag PUT /tags/:id
func PutTag(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	req := tracks.Tag{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	tag, err := jamDB.TagUpdate(uint(id), &req)
	if err == tracks.ErrorNotFound {
		return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound))
	} else if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, tag)
}

// PostTag POST /tags/
func PostTag(ctx echo.Context) error {
	req := tracks.Tag{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	db := jamDB.DB().Save(&req)
	if db.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, db.Error.Error()))
	}

	tag, err := jamDB.Tag(req.ID)
	if err == tracks.ErrorNotFound {
		return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound))
	} else if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusCreated, tag)
}

// Authors GET /authors
func Authors(ctx echo.Context) error {
	t, err := jamDB.Authors()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, t)
}

// Author GET /authors/:id
func Author(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	t, err := jamDB.Author(uint(id))
	if err != nil {
		switch err {
		case tracks.ErrorNotFound:
			return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound, err.Error()))
		default:
			return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
		}
	}

	return ctx.JSON(http.StatusOK, t)
}

// PutAuthor PUT /authors/:id
func PutAuthor(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	req := tracks.Author{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	author, err := jamDB.AuthorUpdate(uint(id), &req)
	if err == tracks.ErrorNotFound {
		return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound))
	} else if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, author)
}

// PostAuthor POST /authors/
func PostAuthor(ctx echo.Context) error {
	req := tracks.Author{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	db := jamDB.DB().Save(&req)
	if db.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, db.Error.Error()))
	}

	author, err := jamDB.Author(req.ID)
	if err == tracks.ErrorNotFound {
		return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound))
	} else if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusCreated, author)
}

// PostTrack POST /tracks
func PostTrack(ctx echo.Context) error {
	file, err := ctx.FormFile("file")
	if err != nil {
		return err
	}

	if !helpers.IsMP3(file.Filename) {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, "bad file type, must be MP3 file"))
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	filePath := path.Join(config.Get().TracksDir, file.Filename)

	// файл с таким именем уже есть - нужно сохранить с другим именем
	for {
		_, err = os.OpenFile(filePath, os.O_RDONLY, 665)
		if err != nil {
			// ошибка - файла нет, всё ОК, можно создавать новый
			break
		}
		var newFileName string
		newFileName, err = helpers.NewFileName(file.Filename)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, "bad file name"))
		}

		filePath = path.Join(config.Get().TracksDir, newFileName)
	}

	// Destination
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	track, err := tracks_sync.ProcessMP3Track(filePath)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusCreated, track)
}

// PutTrack PUT /tracks/:id
func PutTrack(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	req := tracks.Track{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	t, err := jamDB.Track(uint(id))
	if err != nil {
		switch err {
		case tracks.ErrorNotFound:
			return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound, err.Error()))
		default:
			return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
		}
	}

	// set filepath, request not contains it
	req.FilePath = t.FilePath

	err = tracks_sync.UpdateMP3Track(&req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	track, err := jamDB.TrackUpdate(uint(id), &req)
	if err == tracks.ErrorNotFound {
		return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound))
	} else if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, track)
}

// PostPlaylist POST /playlists/
func PostPlaylist(ctx echo.Context) error {
	req := tracks.Playlist{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	db := jamDB.DB().Save(&req)
	if db.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, db.Error.Error()))
	}

	playlist, err := jamDB.Playlist(req.ID)
	if err == tracks.ErrorNotFound {
		return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound))
	} else if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusCreated, playlist)
}

// Playlists GET /playlists/
func Playlists(ctx echo.Context) error {
	playlists, err := jamDB.Playlists()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, playlists)
}

// Playlist GET /playlists/:id
func Playlist(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	playlist, err := jamDB.Playlist(uint(id))
	if err != nil {
		switch err {
		case tracks.ErrorNotFound:
			return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound, err.Error()))
		default:
			return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
		}
	}

	return ctx.JSON(http.StatusOK, playlist)
}

// PutPlaylist PUT /playlists/:id
func PutPlaylist(ctx echo.Context) error {
	idParam := ctx.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	req := tracks.Playlist{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, newError(http.StatusBadRequest, err.Error()))
	}

	playlist, err := jamDB.PlaylistUpdate(uint(id), &req)
	if err == tracks.ErrorNotFound {
		return ctx.JSON(http.StatusNotFound, newError(http.StatusNotFound))
	} else if err != nil {
		return ctx.JSON(http.StatusInternalServerError, newError(http.StatusInternalServerError, err.Error()))
	}

	return ctx.JSON(http.StatusOK, playlist)
}

func newError(code int, message ...string) ErrorResp {
	msg := ""
	if len(message) == 0 || message[0] == "" {
		msg = http.StatusText(code)
	} else {
		msg = message[0]
	}

	return ErrorResp{Error: msg, Code: code}
}

type QueueController struct {
	jm *dj.JamManager
}

// Queue users GET /queue/
func (c QueueController) Users(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, struct {
		Users []string `json:"users"`
	}{
		Users: c.jm.Users(),
	})
}

// Queue command POST /queue/:command
func (c QueueController) Command(ctx echo.Context) error {
	command := ctx.Param("command")
	username := ctx.QueryParam("user")

	msg, err := c.jm.APICommand(command, username)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, struct {
			Error string `json:"error"`
		}{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, struct {
		Message string `json:"message"`
	}{
		Message: msg,
	})
}

// Queue command POST /tts/
func (c QueueController) TTS(ctx echo.Context) error {

	type ttsMessage struct {
		Lang    string `json:"lang"`
		Message string `json:"message"`
	}

	msg := ttsMessage{}

	err := ctx.Bind(&msg)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, struct {
			Error string `json:"error"`
		}{Error: err.Error()})
	}

	c.jm.TextToSpeech(msg.Lang, msg.Message)

	return ctx.JSON(http.StatusOK, struct {
		Message string `json:"message"`
	}{
		Message: "ok",
	})
}
