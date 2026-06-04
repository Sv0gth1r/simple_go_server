package main
import _ "github.com/lib/pq"
import "github.com/joho/godotenv"
import "github.com/Sv0gth1r/simple_go_server/internal/database"
import "github.com/Sv0gth1r/simple_go_server/internal/auth"
import "github.com/google/uuid"
import(
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"strings"
	"database/sql"
	"os"
	"time"
	"errors"
	"context"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
	secret string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func handle_healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Requesting /healtz route\n")
	w.WriteHeader(200)
	n, err := w.Write([]byte(fmt.Sprintf("OK")))
	if err != nil {
		log.Printf("Error while writing: %v\n", err)
	}
	fmt.Printf("%v bytes written.\n", n)
}

func filter_profanity(str string) string {
	s_tab := strings.Split(str, " ")
	for i, word := range s_tab {
		lword := strings.ToLower(word)
		switch lword {
		case "kerfuffle", "sharbert", "fornax":
			s_tab[i] = "****"
		default:
			continue
		}
	}
	return strings.Join(s_tab, " ")
}



func validateChirp(str string) (string, error) {
	var clean_str string
if len(str) > 140 {
		log.Printf("Request Error: Chirp too long")
		err := errors.New("Chirp is too long")
		return clean_str, err
	} else {
		log.Printf("Chirp OK")
		clean_str = filter_profanity(str)
	}
	return clean_str, nil
}

func (cfg *apiConfig)handle_chirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type response struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.NullUUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	headers := r.Header
	params := parameters{}
	res := response{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("Error decoding parameters: %s", err)
		dat, err := json.Marshal(res)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			return
		}
		w.Write(dat)
		return
	}
	token, err := auth.GetBearerToken(headers)
	if err != nil {
		log.Printf("Error fetching the token: %v\n", err)
		return
	}
	id, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		log.Printf("Error validating token: %v\n", err)
		w.WriteHeader(401)
		return
	}
	res.Body, err = validateChirp(params.Body)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("Error while validating chirp: %v\n", err)
		return
	} else {
		chirpParam := database.CreateChirpParams{
			Body : res.Body,
			UserID : uuid.NullUUID{UUID:id, Valid:true,},
		}
		log.Printf("Create chirp with userID: %v\n", chirpParam.UserID)
		chirp, err := cfg.dbQueries.CreateChirp(r.Context(), chirpParam)
		if err != nil {
			w.WriteHeader(500)
			log.Printf("Error creating chirp: %v\n", err)
			return
		}
		res.ID = chirp.ID
		res.UpdatedAt = chirp.UpdatedAt
		res.CreatedAt = chirp.CreatedAt
		res.Body = chirp.Body
		res.UserID = uuid.NullUUID{UUID: id, Valid: true,}
		log.Printf("Respond with user_id: %v\n", id)
	}
	dat, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("Error marshalling JSON: %s", err)
		return
	}
	w.WriteHeader(201)
	w.Write(dat)
	w.Header().Set("Content-Type", "application/json")
	return
}

func (cfg *apiConfig)handle_users(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	res := response{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		w.WriteHeader(500)
		dat, err := json.Marshal(res)
		if err != nil {
			log.Printf("Error marshalling JSON: %s\n", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(dat)
		return
	}
	pwd, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hashing password: %v\n", err)
		return
	}
	userStruct := database.CreateUserParams {
		Email: params.Email,
		HashedPassword: pwd,
	}
	ret, err := cfg.dbQueries.CreateUser(r.Context(), userStruct)
	if err != nil {
		log.Printf("Error creating user: %s\n", err)
		return
	} else {
		res.ID = ret.ID
		res.CreatedAt = ret.CreatedAt
		res.UpdatedAt = ret.UpdatedAt
		res.Email = ret.Email
	}
	dat, err := json.Marshal(res)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		return
	}
	w.WriteHeader(201)
	w.Write(dat)
	w.Header().Set("Content-Type", "application/json")
	return
}

func (cfg *apiConfig)handle_hits(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Requesting /metrics route\n")
	w.WriteHeader(200)
	n, err := w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())))
	if err != nil {
		log.Printf("Error while writing: %v\n", err)
	}
	fmt.Printf("%v bytes written.\n", n)
}

func (cfg *apiConfig)handle_resetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	if cfg.platform != "dev" {
		w.WriteHeader(503)
		return
	}
	err := cfg.dbQueries.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("Error deleting users: %v\n", err)
		return
	}
	w.WriteHeader(200)
	n, err := w.Write([]byte(fmt.Sprintf("OK")))
	if err != nil {
		log.Printf("Error while writing: %v\n", err)
	}
	fmt.Printf("%v bytes written.\n", n)
}

func (cfg *apiConfig)fetch_chirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.NullUUID `json:"user_id"`
	}
	uuid, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		log.Printf("Error while parsing argument chirpID: %v\n", err)
		w.WriteHeader(500)
		return
	}
	log.Printf("Looking for chirp with uuid %v\n", uuid)
	ret, err := cfg.dbQueries.GetChirpById(r.Context(), uuid)
	if err != nil {
		log.Printf("Error fetching chirp: %v\n", err)
		w.WriteHeader(404)
		return
	}
	res := chirp {
		ID: ret.ID,
		CreatedAt: ret.CreatedAt,
		UpdatedAt: ret.UpdatedAt,
		Body: ret.Body,
		UserID: ret.UserID,	
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(res)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		return
	}
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("Error while writing: %v\n", err)
	}
}

func (cfg *apiConfig)delete_chirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error fetching the token: %v\n", err)
		w.WriteHeader(401)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		log.Printf("Error validating token: %v\n", err)
		w.WriteHeader(401)
		return
	}
	uuid, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		log.Printf("Error while parsing argument chirpID: %v\n", err)
		w.WriteHeader(500)
		return
	}
	chirp, err := cfg.dbQueries.GetChirpById(r.Context(), uuid)
	if err != nil {
		log.Printf("Error fetching chirp: %v\n", err)
		return
	}
	if chirp.UserID.UUID != userID {
		log.Printf("Error: User with ID:%v is not author of the chirp (author id: %v)\n", userID, chirp.UserID)
		w.WriteHeader(403)
		return
	}
	err = cfg.dbQueries.DeleteChirpFromChirpId(r.Context(), uuid)
	if err != nil {
		log.Printf("Error fetching chirp: %v\n", err)
		w.WriteHeader(404)
		return
	}
	w.WriteHeader(204)
	return
}

func (cfg *apiConfig)fetch_chirps(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.NullUUID `json:"user_id"`
	}
	type response struct {
		Chirps []chirp
	}
	chirps, err := cfg.dbQueries.GetAllChirpsOrderedByCreationDate(r.Context())
	if err != nil {
		log.Printf("Error fetching chirps: %v\n", err)
		w.WriteHeader(500)
		return
	}
	res := response{}
	for _, oneChirp := range chirps {
		tmpChirp := chirp {
			ID: oneChirp.ID,
			CreatedAt: oneChirp.CreatedAt,
			UpdatedAt: oneChirp.UpdatedAt,
			Body: oneChirp.Body,
			UserID: oneChirp.UserID,
		}
		res.Chirps = append(res.Chirps, tmpChirp)
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(res.Chirps)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		return
	}
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("Error while writing: %v\n", err)
	}
}

func (cfg *apiConfig)handle_login(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	type response struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	res := response{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		w.WriteHeader(500)
		dat, err := json.Marshal(res)
		if err != nil {
			log.Printf("Error marshalling JSON: %s\n", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(dat)
		return
	}
	user, err := cfg.dbQueries.GetUserFromEmail(r.Context(), params.Email)
	if err != nil {
		log.Printf("Error fetching user: %v\n", err)
		return
	}
	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		log.Printf("Error checking password against hash: %v\n", err)
		return
	}
	if match == true {
		token, err := auth.MakeJWT(user.ID, cfg.secret)
		if err != nil {
			log.Printf("Error creating Token: %v\n", err)
			return
		}
		refresh_token := database.CreateRefreshTokenParams {
			Token: auth.MakeRefreshToken(),
			UserID: user.ID,
			ExpiresAt: time.Now().UTC().Add(time.Hour * 24 * 60),
		}
		_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), refresh_token)
		if err != nil {
			log.Printf("Error creating refresh_token in db: %v\n", err)
			return
		}
		res := response{
			ID: user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email: user.Email,
			Token: token,
			RefreshToken: refresh_token.Token,
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		dat, err := json.Marshal(res)
		if err != nil {
			log.Printf("Error marshalling JSON: %s\n", err)
			return
		}
		_, err = w.Write(dat)
		if err != nil {
			log.Printf("Error while writing: %v\n", err)
		}
	} else {
		w.WriteHeader(401)
		_, err := w.Write([]byte(fmt.Sprintf("Incorrect email or password")))
		if err != nil {
			log.Printf("Error while writing: %v\n", err)
		}
	}
	return
}

func (cfg *apiConfig)refresh_token(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting bearer token: %v\n", err)
		return
	}
	tokenStruct, err := cfg.dbQueries.GetRefreshToken(context.Background(), token)
	if err != nil {
		log.Printf("Error fetching refresh token %v: %v\n", token, err)
		w.WriteHeader(401)
		return
	}
	if tokenStruct.RevokedAt.Valid == true {
		log.Printf("token %v is revoked\n", token)
		w.WriteHeader(401)
		return
	}
	jwToken, err := auth.MakeJWT(tokenStruct.UserID, cfg.secret)
	res := response{
		Token: jwToken,
	}
	w.WriteHeader(200)
	dat, err := json.Marshal(res)
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("Error while writing: %v\n", err)
	}
	return
}

func (cfg *apiConfig)revoke_token(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting bearer token: %v\n", err)
		return
	}
	_, err = cfg.dbQueries.GetRefreshToken(context.Background(), token)
	if err != nil {
		log.Printf("Error fetching refresh token %v: %v\n", token, err)
		w.WriteHeader(401)
		return
	}
	cfg.dbQueries.RevokeToken(context.Background(), token)
	w.WriteHeader(204)
	return
}

func (cfg *apiConfig)update_user(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		NewPassword string `json:"password"`
		NewEmail string `json:"email"`
	}
	type response struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error fetching the token: %v\n", err)
		w.WriteHeader(401)
		return
	}
	id, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		log.Printf("Error validating token: %v\n", err)
		w.WriteHeader(401)
		return
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		w.WriteHeader(500)
		return
	}
	pwd, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		log.Printf("Error hashing password: %v\n", err)
		return
	}
	pwdUpdateStruct := database.EditUserPwdParams {
		ID: id,
		HashedPassword: pwd,
	}
	err = cfg.dbQueries.EditUserPwd(r.Context(), pwdUpdateStruct)
	if err != nil {
		log.Printf("Error updating password: %s\n", err)
		return
	}
	mailUpdateStruct := database.EditUserEmailParams {
		ID: id,
		Email: params.NewEmail,
	}
	err = cfg.dbQueries.EditUserEmail(r.Context(), mailUpdateStruct)
	if err != nil {
		log.Printf("Error updating email: %s\n", err)
		return
	}
	user, err := cfg.dbQueries.GetUserFromEmail(r.Context(), params.NewEmail)
	if err != nil {
		log.Printf("Error fetching user: %v\n", err)
		return
	}
	res := response{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(res)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		return
	}
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("Error while writing: %v\n", err)
	}
	return
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error opening database: %v\n", err)
		return
	}
	serveMux := http.NewServeMux()
	var server http.Server
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	var apiCfg apiConfig
	apiCfg.dbQueries = database.New(db)
	apiCfg.platform = os.Getenv("PLATFORM")
	apiCfg.secret = os.Getenv("SECRET")

	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	serveMux.HandleFunc("GET /api/healthz", handle_healthz)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handle_hits)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handle_resetHits)
	serveMux.HandleFunc("POST /api/users", apiCfg.handle_users)
	serveMux.HandleFunc("POST /api/login", apiCfg.handle_login)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.handle_chirps)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.fetch_chirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.fetch_chirp)
	serveMux.HandleFunc("POST /api/refresh", apiCfg.refresh_token)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.revoke_token)
	serveMux.HandleFunc("PUT /api/users", apiCfg.update_user)
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.delete_chirp)
	server.Addr = ":8080"
	server.Handler = serveMux

	// Start the server
	fmt.Println("Start server on port 8080")
	log.Fatal(server.ListenAndServe())
}
