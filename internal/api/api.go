package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/herrmannplatz/chirpy/internal/database"
	"github.com/herrmannplatz/chirpy/pkg/auth"
	"github.com/herrmannplatz/chirpy/pkg/util"
)

type responseError struct {
	Error string `json:"error"`
}

type Config struct {
	FileserverHits atomic.Int32
	Db             *database.Queries
	Platform       string
	Secret         string
	POLKA_KEY      string
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type User struct {
	Id            uuid.UUID `json:"id"`
	Created_at    time.Time `json:"created_at"`
	Updated_at    time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	Is_chirpy_red bool      `json:"is_chirpy_red"`
}

func (cfg *Config) HandlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := cfg.Db.DeleteUsers(r.Context()); err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Users deleted"))
}

func (cfg *Config) HandlerPostUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type responseData struct {
		Id            uuid.UUID `json:"id"`
		Created_at    time.Time `json:"created_at"`
		Updated_at    time.Time `json:"updated_at"`
		Email         string    `json:"email"`
		Is_chirpy_red bool      `json:"is_chirpy_red"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: "Error parsing payload"})
		return
	}

	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	user, err := cfg.Db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashed_password,
	})

	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	util.SendJSON(w, http.StatusCreated, responseData{
		Id:            user.ID,
		Created_at:    user.CreatedAt,
		Updated_at:    user.UpdatedAt,
		Email:         user.Email,
		Is_chirpy_red: user.IsChirpyRed,
	})
}

func (cfg *Config) HandlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type responseData struct {
		Id            uuid.UUID `json:"id"`
		Created_at    time.Time `json:"created_at"`
		Updated_at    time.Time `json:"updated_at"`
		Email         string    `json:"email"`
		Token         string    `json:"token"`
		Refresh_token string    `json:"refresh_token"`
		Is_chirpy_red bool      `json:"is_chirpy_red"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: "Error parsing payload"})
		return
	}

	user, err := cfg.Db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	if err := auth.CheckPasswordHash(params.Password, user.HashedPassword); err != nil {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: "Incorrect email or password"})
		return
	}

	refresh_token, err := auth.MakeRefreshToken()
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	_, err = cfg.Db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refresh_token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	if err := auth.CheckPasswordHash(params.Password, user.HashedPassword); err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.Secret, time.Hour)
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	util.SendJSON(w, http.StatusOK, responseData{
		Id:            user.ID,
		Created_at:    user.CreatedAt,
		Updated_at:    user.UpdatedAt,
		Email:         user.Email,
		Is_chirpy_red: user.IsChirpyRed,
		Token:         token,
		Refresh_token: refresh_token,
	})
}

func (cfg *Config) HandlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	type responseData struct {
		Token string `json:"token"`
	}

	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: err.Error()})
		return
	}
	token, err := cfg.Db.GetRefreshTokenByToken(r.Context(), refresh_token)
	if err != nil {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: err.Error()})
		return
	}
	if token.RevokedAt.Valid {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: "Refresh token revoked"})
		return
	}
	if token.ExpiresAt.Before(time.Now()) {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: "Refresh token expired"})
		return
	}
	new_token, err := auth.MakeJWT(token.UserID, cfg.Secret, time.Hour)
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}
	util.SendJSON(w, http.StatusOK, responseData{Token: new_token})
}

func (cfg *Config) HandlerPutUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type responseData struct {
		Id            uuid.UUID `json:"id"`
		Created_at    time.Time `json:"created_at"`
		Updated_at    time.Time `json:"updated_at"`
		Email         string    `json:"email"`
		Is_chirpy_red bool      `json:"is_chirpy_red"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: "Error parsing payload"})
		return
	}

	user_id, err := auth.ValidateRequestToken(r, cfg.Secret)
	if err != nil {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: err.Error()})
		return
	}

	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	user, err := cfg.Db.UpdateEmailAndPassword(r.Context(), database.UpdateEmailAndPasswordParams{
		ID:             user_id,
		Email:          params.Email,
		HashedPassword: hashed_password,
	})

	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
	}

	util.SendJSON(w, http.StatusOK, responseData{
		Id:            user.ID,
		Created_at:    user.CreatedAt,
		Updated_at:    user.UpdatedAt,
		Email:         user.Email,
		Is_chirpy_red: user.IsChirpyRed,
	})
}

func (cfg *Config) HandlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: err.Error()})
		return
	}
	tkn, err := cfg.Db.GetRefreshTokenByToken(r.Context(), refresh_token)
	if tkn.RevokedAt.Valid {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: err.Error()})
		return
	}
	err = cfg.Db.RevokeRefreshToken(r.Context(), refresh_token)
	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}
	util.SendJSON(w, http.StatusNoContent, nil)
}

func (cfg *Config) HandlerPostChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	user_id, err := auth.ValidateRequestToken(r, cfg.Secret)
	if err != nil {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: err.Error()})
		return
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: "Error parsing payload"})
		return
	}

	if (len(params.Body) == 0) || (len(params.Body) > 140) {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: "Invalid chirp length"})
		return
	}

	replacer := strings.NewReplacer(
		"kerfuffle", "****",
		"sharbert", "****",
		"fornax", "****",
		"Fornax", "****",
	)

	chirp, err := cfg.Db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   replacer.Replace(params.Body),
		UserID: user_id,
	})

	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	util.SendJSON(w, http.StatusCreated, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
}

func (cfg *Config) HandlerGetChirps(w http.ResponseWriter, r *http.Request) {
	authorID, _ := uuid.Parse(r.URL.Query().Get("author_id"))

	sortDirection := "asc"
	if r.URL.Query().Get("sort") == "desc" {
		sortDirection = "desc"
	}

	chirps, err := cfg.Db.GetChirpsWithOptions(r.Context(), database.GetChirpsWithOptionsParams{
		UserID:  authorID,
		Column2: sortDirection,
	})

	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
	}

	response := []Chirp{}
	for _, chirp := range chirps {
		response = append(response, Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			UserID:    chirp.UserID,
			Body:      chirp.Body,
		})
	}

	sort.Slice(response, func(i, j int) bool {
		if sortDirection == "desc" {
			return chirps[i].CreatedAt.After(response[j].CreatedAt)
		}
		return chirps[i].CreatedAt.Before(response[j].CreatedAt)
	})

	util.SendJSON(w, http.StatusOK, response)
}

func (cfg *Config) HandlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: "Invalid chirp id"})
		return
	}

	user_id, err := auth.ValidateRequestToken(r, cfg.Secret)
	if err != nil {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: err.Error()})
		return
	}

	chirp, err := cfg.Db.GetChirp(r.Context(), id)
	if err != nil {
		util.SendJSON(w, http.StatusNotFound, responseError{Error: err.Error()})
		return
	}
	if chirp.UserID != user_id {
		util.SendJSON(w, http.StatusForbidden, responseError{Error: "You do not have permission to delete this chirp"})
		return
	}

	err = cfg.Db.DeleteChirp(r.Context(), database.DeleteChirpParams{
		ID:     id,
		UserID: user_id,
	})

	if err != nil {
		util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
		return
	}

	util.SendJSON(w, http.StatusNoContent, nil)
}

func (cfg *Config) HandlerGetChirp(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: "Invalid chirp id"})
		return
	}

	chirp, err := cfg.Db.GetChirp(r.Context(), id)
	if err != nil {
		util.SendJSON(w, http.StatusNotFound, responseError{Error: err.Error()})
		return
	}

	util.SendJSON(w, http.StatusOK, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
}

func (cfg *Config) HandlerPolkaWebhook(w http.ResponseWriter, r *http.Request) {
	type polkaEvent struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	var event polkaEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		util.SendJSON(w, http.StatusBadRequest, responseError{Error: err.Error()})
		return
	}

	apiKey := auth.GetAPIKey(r.Header)
	if apiKey != cfg.POLKA_KEY {
		util.SendJSON(w, http.StatusUnauthorized, responseError{Error: "Invalid API key"})
		return
	}

	if event.Event != "user.upgraded" {
		util.SendJSON(w, http.StatusNoContent, nil)
		return
	}

	err := cfg.Db.UpdateIsChirpyRed(r.Context(), database.UpdateIsChirpyRedParams{
		IsChirpyRed: true,
		ID:          event.Data.UserID,
	})
	if err != nil {
		util.SendJSON(w, http.StatusNotFound, responseError{Error: err.Error()})
		return
	}
	util.SendJSON(w, http.StatusNoContent, nil)
}
