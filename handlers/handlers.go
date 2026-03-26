package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/internal/auth"
	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/logging"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"github.com/shahanmmiah/ravelpin/services"
)

const POSTAMOUNT = 6
const REFRESHTOKENCOOKIE = "rpRefreshToken"
const JWTTOKENCOOKIE = "rpToken"

const DEFAULTREFRESHEXPIRY = time.Duration(time.Duration(3600) * time.Second)
const DEFAULTJWTEXPIRY = time.Duration(time.Duration(960) * time.Second)

// BASE HANDLERS

func HandleHandler(mux *http.ServeMux, handle http.Handler, endpoint, mthd string) error {

	if endpoint == "" {
		return fmt.Errorf("%v Ns cannot be empty", endpoint)
	}
	if mthd == "" {
		return fmt.Errorf("%v Method cannot be empty", mthd)
	}

	mux.Handle(fmt.Sprintf("%s %s", mthd, endpoint), handle)

	return nil
}

func MiddleWareServeFile(file string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		http.ServeFile(resp, req, file)
	})
}

func HandlerGetLog() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(fmt.Sprintf("Unable to parse form %s", err.Error())))
			return
		}

		file := req.Form.Get("logFile")

		if file != "" {
			log, err := logging.GetLog(file)

			if err != nil {
				resp.WriteHeader(http.StatusInternalServerError)
				resp.Write([]byte(fmt.Sprintf("Unable to load log %s", err.Error())))
				return
			}

			slog.InfoContext(req.Context(), fmt.Sprintf("log file content : %v", log))

			logHTML := components.Log(log)
			w, err := MarhalComponent(logHTML)
			if err != nil {
				resp.WriteHeader(http.StatusInternalServerError)
				resp.Write([]byte(fmt.Sprintf("Unable to Marshal log %s", err.Error())))

				return

			}

			slog.InfoContext(req.Context(), fmt.Sprintf("log file content : %v", w.String()))

			resp.WriteHeader(http.StatusOK)
			resp.Header().Set("Content-Type", "text/html")
			resp.Write(w.Bytes())
			return

		}

		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte(fmt.Sprintf("cannot find log %s", err.Error())))

	})
}

func MarhalComponent(htmlComponents templ.Component) (*bytes.Buffer, error) {
	w := new(bytes.Buffer)

	err := htmlComponents.Render(context.Background(), w)
	if err != nil {
		return nil, err

	}

	return w, nil

}

func (cfg *ApiConfig) HandlerGetLogs() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		_, err := cfg.CheckJwtToken(req)

		if err != nil {
			cfg.HandlerRefresh().ServeHTTP(resp, req)
		}

		dir, err := os.ReadDir(os.Getenv("LOGPATH"))
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte("could not find save logs"))
			return
		}

		ends := make([]string, 0)
		for _, file := range dir {
			ends = append(ends, file.Name())
		}

		htmlComponents := components.LogList(ends)

		w, err := MarhalComponent(htmlComponents)

		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not render html components, %v", err.Error()))
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return
		}

		resp.WriteHeader(http.StatusOK)
		resp.Header().Set("Content-Type", "text/html")
		resp.Write(w.Bytes())

	})
}

func MiddleWareModelPreload(model *recoginition.ClassifyModel, mdl func(string, *recoginition.ClassifyModel) ([]string, error)) func(string) ([]string, error) {
	return func(s string) ([]string, error) {
		return mdl(s, model)
	}
}

func (cfg *ApiConfig) HandlerRefresh() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		refreshCookie, err := req.Cookie(JWTTOKENCOOKIE)
		if err != nil {

			errMsg := fmt.Sprintf("no Refresh token found, %v", err.Error())
			slog.ErrorContext(req.Context(), errMsg)

			http.Redirect(resp, req, "./login", http.StatusSeeOther)

			return
		}

		refreshToken, err := cfg.Db.GetRefreshToken(req.Context(), refreshCookie.Value)
		if err != nil {

			errMsg := fmt.Sprintf("Refresh token Error, %v", err.Error())
			slog.ErrorContext(req.Context(), errMsg)

			//resp.WriteHeader(http.StatusUnauthorized)
			resp.Header().Set("Content-Type", "text/plain")

			http.Redirect(resp, req, "/login", http.StatusSeeOther)
			return
		}

		if time.Now().After(refreshToken.ExpiresAt) || refreshToken.RevokedAt.Valid {
			errMsg := fmt.Sprintf("Refresh token Expired, %v", err.Error())
			slog.ErrorContext(req.Context(), errMsg)

			//resp.WriteHeader(http.StatusUnauthorized)
			resp.Header().Set("Content-Type", "text/plain")

			http.Redirect(resp, req, "/login", http.StatusSeeOther)
			return

		}

		jwtCookie, err := req.Cookie(JWTTOKENCOOKIE)
		if err != nil {
			errMsg := fmt.Sprintf("no Auth token found, %v", err.Error())
			slog.ErrorContext(req.Context(), errMsg)

			//resp.WriteHeader(http.StatusUnauthorized)
			resp.Header().Set("Content-Type", "text/hmtl")

			http.Redirect(resp, req, "/login", http.StatusSeeOther)

			return
		}

		_, err = auth.ValidateJWT(jwtCookie.Value, os.Getenv("TOKENSECRET"))

		if err != nil {
			_, err := cfg.SetJwtToken(refreshToken.UserID, &resp)

			if err != nil {
				errMsg := fmt.Sprintf("Error refreshing auth token, %v", err.Error())
				slog.ErrorContext(req.Context(), errMsg)

				//resp.WriteHeader(http.StatusInternalServerError)
				resp.Header().Set("Content-Type", "text/plain")

				http.Redirect(resp, req, "/login", http.StatusSeeOther)
				return

			}

		}

		resp.WriteHeader(http.StatusNoContent)

	})

}

// RAVELRY HANDLERS

func (cfg *ApiConfig) MiddleWareGetRavelLink(clasifyModel *recoginition.ClassifyModel) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		_, err := cfg.CheckJwtToken(req)

		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("jwt error %v", err.Error()))
			cfg.HandlerRefresh().ServeHTTP(resp, req)
			return
		}

		requestId := uuid.New()

		logObj, err := logging.MakeLoggerObject(requestId)
		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not log request, %v", err.Error()))

			resp.WriteHeader(http.StatusInternalServerError)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		logger := slog.New(logObj.LogHandler)
		slog.SetDefault(logger)
		err = req.ParseForm()
		if err != nil {
			slog.InfoContext(req.Context(), fmt.Sprintf("could not parse form, %v", err.Error()))
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		pinLink := string(req.PostForm.Get("pinlink"))

		slog.InfoContext(req.Context(), fmt.Sprintf("looking at pinlink %v", pinLink))
		ravelPatterns, err := HandlerFindRavelFromPins(pinLink, MiddleWareModelPreload(clasifyModel, services.ImageClasify))

		if err != nil {
			slog.InfoContext(req.Context(), fmt.Sprintf("could not find ravelry links, %v", err.Error()))

			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		htmlComponents := components.RavelPosts(ravelPatterns)
		w, err := MarhalComponent(htmlComponents)

		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not render html components, %v", err.Error()))

			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		resp.WriteHeader(http.StatusOK)
		resp.Header().Set("Content-Type", "text/html")
		resp.Write(w.Bytes())

		slog.InfoContext(req.Context(), "Request Complete", slog.String("URI", req.RequestURI), slog.String("input", req.FormValue("pinlink")))
		logObj.CloseLogFile()

	})
}

func HandlerFindRavelFromPins(link string, clasifyCommand func(string) ([]string, error)) ([]services.RavelryPattern, error) {

	pinLink, err := services.GetPinImageLink(link)

	if err != nil {
		return []services.RavelryPattern{}, err
	}

	searchQueries, err := clasifyCommand(link)

	ravPatterns, err := GetRavelPatterns(searchQueries)

	if err != nil {
		return []services.RavelryPattern{}, err
	}

	if len(ravPatterns) > 0 {
		slog.Info("comparing found ravelry posts to pintrest image...", slog.Any("postFound", ravPatterns))
		bestMatchPatterns, err := services.GetBestsImages(ravPatterns, pinLink, POSTAMOUNT)

		if err != nil {
			return []services.RavelryPattern{}, err
		}

		//ravelURL := fmt.Sprintf("%v%s", os.Getenv("RAVELURL"), bestMatchPattern.Permalink)

		//fmt.Printf("closest match found: %s", ravelURL)
		//replCommand.OpenURL(ravelURL)
		return bestMatchPatterns, nil
	}

	return []services.RavelryPattern{}, fmt.Errorf("no ravelry post found")
}

func GetRavelPatterns(queries []string) ([]services.RavelryPattern, error) {
	godotenv.Load()
	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := os.Getenv("RAVELPATTERNSEARCH")

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		slog.Error(fmt.Sprintf("client: could not create request %v", err.Error()))
		os.Exit(1)
	}
	params := req.URL.Query()

	queryString := ""
	for num, query := range queries {
		if num == 0 {
			queryString += fmt.Sprintf("%s", query)
		} else {
			queryString += fmt.Sprintf("&%s", query)
		}

		if num == 0 {
		}

	}
	params.Add("query", queryString)
	params.Add("page_size", "10")
	//params.Add("weight", "cobweb")

	req.URL.RawQuery = params.Encode()

	slog.InfoContext(req.Context(), fmt.Sprintf("params to search : %v", params.Encode()))

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(APIUS, APIKEY)
	//req.Header.Set("Authorization", "<access_token>")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return []services.RavelryPattern{}, err

	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	jsonData := make(map[string]interface{}, 0)

	//fmt.Println(string(data))

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return []services.RavelryPattern{}, err

	}

	patterMap, _ := jsonData["patterns"].([]any)
	patterns := make([]services.RavelryPattern, 0)
	for _, items := range patterMap {

		pattern := services.RavelryPattern{}

		patternData, err := json.Marshal(items)
		if err != nil {
			return []services.RavelryPattern{}, err

		}

		json.Unmarshal(patternData, &pattern)

		patterns = append(patterns, pattern)

	}

	return patterns, nil

}

// USER CREATE HANDLERS

func (cfg *ApiConfig) HandlerVerify() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		vals := req.URL.Query()

		verCode := vals.Get("vercode")
		if verCode == "" {
			headers := map[string]string{"Content-Type": "text/plain"}
			ErrorMsg(&resp, req, http.StatusBadRequest, fmt.Sprintf("No verify code param found"), headers)
			return
		}

		email := vals.Get("email")
		if email == "" {
			errMsg := fmt.Sprintf("no email param found")
			headers := map[string]string{"Content-Type": "text/plain"}
			ErrorMsg(&resp, req, http.StatusBadRequest, errMsg, headers)
			return
		}

		passHash := vals.Get("passhash")
		if passHash == "" {
			errMsg := fmt.Sprintf("no pass param found")
			headers := map[string]string{"Content-Type": "text/plain"}
			ErrorMsg(&resp, req, http.StatusBadRequest, errMsg, headers)
			return
		}

		verData, err := cfg.Db.GetVerifyTokenFromId(req.Context(), email)

		if err != nil {
			errMsg := fmt.Sprintf("error with getting verify token, %v", err.Error())
			headers := map[string]string{"Content-Type": "text/plain"}
			ErrorMsg(&resp, req, http.StatusBadRequest, errMsg, headers)
			return
		}

		if verData.ExpiresAt.Before(time.Now()) {
			errMsg := fmt.Sprintf("verify code has expired")
			headers := map[string]string{"Content-Type": "text/plain"}
			cfg.Db.DeleteVerifyTokenFromEmail(req.Context(), email)
			ErrorMsg(&resp, req, http.StatusBadRequest, errMsg, headers)
			return
		}

		_, err = cfg.Db.CreateUser(req.Context(), database.CreateUserParams{
			ID:             uuid.New(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			Email:          email,
			HashedPassword: passHash,
		})

		if err != nil {
			errMsg := fmt.Sprintf("error with getting verify token, %v", err.Error())
			headers := map[string]string{"Content-Type": "text/plain"}

			ErrorMsg(&resp, req, http.StatusInternalServerError, errMsg, headers)
			return
		}

		err = cfg.Db.DeleteVerifyTokenFromEmail(req.Context(), email)

		if err != nil {
			errMsg := fmt.Sprintf("error removing used verify tokens, %v", err.Error())
			headers := map[string]string{"Content-Type": "text/plain"}

			ErrorMsg(&resp, req, http.StatusInternalServerError, errMsg, headers)
			return
		}

		resp.Header().Set("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusOK)

		resp.Write([]byte("Succesfully Verified Account :)"))
		return
	})
}

func (cfg *ApiConfig) HandlerSignup() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		headers := map[string]string{"Content-Type": "text/plain"}

		err := req.ParseForm()

		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(fmt.Sprintf("failed parsing form %v", err.Error())))
			return
		}

		email := req.Form.Get("email")

		user, err := cfg.Db.GetUserFromEmail(req.Context(), email)

		if err == nil || user.Email == email {

			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(fmt.Sprintf("account already registered with email")))
			return

		}

		_, err = cfg.Db.GetVerifyTokenFromId(req.Context(), email)

		if err == nil {

			errMsg := "account already registered with email"
			ErrorMsg(&resp, req, http.StatusBadRequest, errMsg, headers)
			return

		}

		password := req.Form.Get("password")
		if password == "" {
			errMsg := fmt.Sprintf("Error gettting password", err.Error())
			ErrorMsg(&resp, req, http.StatusBadRequest, errMsg, headers)
			return

		}

		token, _ := auth.MakeToken(4)

		tokenData, err := cfg.Db.CreateVerifyToken(req.Context(), database.CreateVerifyTokenParams{
			Token:     token,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Email:     email,
			ExpiresAt: time.Now().Add(DEFAULTJWTEXPIRY),
		})

		if err != nil {
			errMsg := fmt.Sprintf("error creating verify token, %s", err.Error())
			ErrorMsg(&resp, req, http.StatusInternalServerError, errMsg, headers)
			return

		}
		slog.Info(fmt.Sprintf("verify token is %s", tokenData.Token))

		passHash, err := auth.HashPassword(password)
		if passHash == "" {
			errMsg := fmt.Sprintf("error making password hash")
			ErrorMsg(&resp, req, http.StatusInternalServerError, errMsg, headers)
			return

		}

		domain, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			domain = req.Host
		}

		err = sendVerifyEmail(email, passHash, token, domain)

		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Header().Set("Content-Type", "text/plain")
			resp.Write([]byte(fmt.Sprintf("Error sending verify email %s", err.Error())))

		}

		sendMsg := fmt.Sprintf("Verify email has been sent to %s", email)
		slog.Info(sendMsg)
		resp.WriteHeader(http.StatusAccepted)
		resp.Header().Set("Content-Type", "text/plain")
		resp.Write([]byte(sendMsg))

	})

}

func (cfg *ApiConfig) HandlerSignupPage() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		htmlComponents, err := MarhalComponent(components.SignupPage())
		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not render html components, %v", err.Error()))
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return
		}

		resp.Write(htmlComponents.Bytes())
	})
}

// LOGIN HANDLERS

func (cfg *ApiConfig) HandlerLogout() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		refreshCookie, err := req.Cookie(REFRESHTOKENCOOKIE)
		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not find cookies %v", err.Error()))
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return
		}

		cfg.Db.RevokeRefreshToken(
			req.Context(),
			database.RevokeRefreshTokenParams{Token: refreshCookie.Value,
				RevokedAt: sql.NullTime{Time: time.Now(), Valid: true}})

		UnsetTokens(&resp)
		resp.Header().Set("HX-Redirect", "/")
		resp.WriteHeader(http.StatusNoContent)

	})
}

func (cfg *ApiConfig) HandlerLoginPage() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		htmlComponents, err := MarhalComponent(components.LoginPage())
		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not render html components, %v", err.Error()))
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return
		}

		resp.Write(htmlComponents.Bytes())
	})
}

func (cfg *ApiConfig) HandlerLogin() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(fmt.Sprintf("failed parsing form %v", err.Error())))
			return
		}

		email := req.Form.Get("email")
		password := req.Form.Get("password")

		usr, err := cfg.Db.GetUserFromEmail(req.Context(), email)

		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(fmt.Sprintf("error getting user %v", err.Error())))
			return
		}

		err = auth.CheckPasswordHash(password, usr.HashedPassword)

		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(fmt.Sprintf("error verifying email %v", err.Error())))
			return
		}

		refreshCookie, err := req.Cookie(REFRESHTOKENCOOKIE)
		if err == nil {
			cfg.Db.RevokeRefreshToken(
				req.Context(),
				database.RevokeRefreshTokenParams{Token: refreshCookie.Value,
					RevokedAt: sql.NullTime{Time: time.Now(), Valid: true}})
		}

		_, _, err = cfg.SetTokens(usr.ID, req, &resp)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Header().Set("Content-Type", "text/plain")
			resp.Write([]byte(fmt.Sprintf("error verifying email %v", err.Error())))
			return
		}

		http.Redirect(resp, req, "/", http.StatusSeeOther)

	})
}

// UTILS

func ErrorMsg(resp *http.ResponseWriter, req *http.Request, status int, errorMsg string, headers map[string]string) {

	slog.InfoContext(req.Context(), errorMsg)
	(*resp).WriteHeader(http.StatusBadRequest)

	for key, vals := range headers {
		(*resp).Header().Set(key, vals)
	}

	(*resp).Write([]byte(errorMsg))
}

func (cfg *ApiConfig) LimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		if !cfg.Serv.RateLimit.GetClientRateLimit(ip) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too Many Requests"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) CheckRefreshTokenExpired(token string) (bool, error) {

	tokenData, err := cfg.Db.GetRefreshToken(context.Background(), token)
	if err != nil {
		return false, fmt.Errorf("error quering refresh token: %v", err.Error())
	}

	expires := time.Now().Compare(tokenData.ExpiresAt)
	if expires > 0 {
		cfg.Db.RevokeRefreshToken(context.Background(),
			database.RevokeRefreshTokenParams{
				Token:     token,
				RevokedAt: sql.NullTime{Time: time.Now(), Valid: true}})
		return true, nil
	}

	return false, nil
}

func UnsetTokens(resp *http.ResponseWriter) {
	refreshcookie := http.Cookie{
		Name:     REFRESHTOKENCOOKIE,
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Now().Add(-100 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(*resp, &refreshcookie)

	jwtcookie := http.Cookie{
		Name:     JWTTOKENCOOKIE,
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Now().Add(-100 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(*resp, &jwtcookie)

}

func (cfg *ApiConfig) SetTokens(userID uuid.UUID, req *http.Request, resp *http.ResponseWriter) (string, string, error) {

	refreshToken, err := cfg.SetRefreshToken(userID, req, resp)
	if err != nil {
		return "", "", err
	}

	jwtToken, err := cfg.SetJwtToken(userID, resp)
	if err != nil {
		return "", "", err
	}

	return refreshToken, jwtToken, nil

}

func (cfg *ApiConfig) SetRefreshToken(userID uuid.UUID, req *http.Request, resp *http.ResponseWriter) (string, error) {

	refreshToken, err := auth.MakeToken(32)
	if err != nil {
		return "", err
	}

	cfg.Db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(DEFAULTREFRESHEXPIRY),
	})
	refreshcookie := http.Cookie{
		Name:     REFRESHTOKENCOOKIE,
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(*resp, &refreshcookie)

	return refreshToken, nil

}

func (cfg *ApiConfig) SetJwtToken(userID uuid.UUID, resp *http.ResponseWriter) (string, error) {

	jwtToken, err := auth.MakeJWT(userID, os.Getenv("TOKENSECRET"), DEFAULTJWTEXPIRY)
	if err != nil {
		return "", err
	}

	jwtcookie := http.Cookie{
		Name:     JWTTOKENCOOKIE,
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(*resp, &jwtcookie)

	return jwtToken, nil

}

func (cfg *ApiConfig) CheckJwtToken(req *http.Request) (uuid.UUID, error) {

	jwtCookie, err := req.Cookie(JWTTOKENCOOKIE)
	if err != nil {
		errMsg := fmt.Sprintf("no Auth token found, %v", err.Error())
		slog.ErrorContext(req.Context(), errMsg)

		return uuid.Nil, err
	}

	userId, err := auth.ValidateJWT(jwtCookie.Value, os.Getenv("TOKENSECRET"))

	if err != nil {
		errMsg := fmt.Sprintf("Auth token invalid, %v", err.Error())
		slog.ErrorContext(req.Context(), errMsg)
		return uuid.Nil, err
	}

	return userId, nil

}

func (cfg *ApiConfig) HandlerHomePage() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		login := true

		if _, err := cfg.CheckJwtToken(req); err == nil {
			login = false
			slog.InfoContext(req.Context(), "user is logged in")
		}

		component := components.HomePage(login)

		w, err := MarhalComponent(component)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte(fmt.Sprintf("Unable to Marshal log %s", err.Error())))

			return

		}
		resp.WriteHeader(http.StatusOK)
		resp.Write(w.Bytes())

	})
}

// SERVER HANDLERS

func (cfg *ApiConfig) SetupServer(clasifyModel *recoginition.ClassifyModel) {

	type handlerItem struct {
		Endpoint string
		Method   string
		Handler  http.Handler
	}

	var handlerItems = []handlerItem{
		{
			Endpoint: "/{$}",
			Method:   "GET",
			Handler:  cfg.HandlerHomePage(),
		},
		{
			Endpoint: "/assets/js/htmx.min.js",
			Method:   "GET",
			Handler:  MiddleWareServeFile("./assets/js/htmx.min.js"),
		},
		{
			Endpoint: "/link",
			Method:   "POST",
			Handler:  cfg.LimitMiddleware(cfg.MiddleWareGetRavelLink(clasifyModel)),
		},
		{
			Endpoint: "/logs",
			Method:   "GET",
			Handler:  cfg.HandlerGetLogs(),
		},
		{
			Endpoint: "/log",
			Method:   "POST",
			Handler:  cfg.LimitMiddleware(HandlerGetLog()),
		},
		{
			Endpoint: "/login",
			Method:   "POST",
			Handler:  cfg.HandlerLogin(),
		},
		{
			Endpoint: "/login",
			Method:   "GET",
			Handler:  cfg.HandlerLoginPage(),
		},
		{
			Endpoint: "/logout",
			Method:   "POST",
			Handler:  cfg.HandlerLogout(),
		},
		{
			Endpoint: "/refresh",
			Method:   "POST",
			Handler:  cfg.HandlerRefresh(),
		},
		{
			Endpoint: "/signup",
			Method:   "GET",
			Handler:  cfg.HandlerSignupPage(),
		},
		{
			Endpoint: "/signup",
			Method:   "POST",
			Handler:  cfg.HandlerSignup(),
		},
		{
			Endpoint: "/verify",
			Method:   "GET",
			Handler:  cfg.HandlerVerify(),
		},
	}

	for _, h := range handlerItems {
		err := HandleHandler(cfg.Serv.Mux, h.Handler, h.Endpoint, h.Method)
		if err != nil {
			slog.ErrorContext(
				context.Background(),
				fmt.Sprintf("error %s handler for %s  %s", h.Method, h.Endpoint, err.Error()))
		}
	}

	server := http.Server{
		Handler: cfg.Serv.Mux,
		Addr:    ":3000",
	}

	slog.Info("Listening on :3000")

	server.ListenAndServe()

}
