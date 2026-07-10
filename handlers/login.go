package handlers

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/internal/auth"
	"github.com/shahanmmiah/ravelpin/internal/database"
)

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
			errMsg := fmt.Sprintf("Error gettting password: %v", err.Error())
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

func (cfg *ApiConfig) HandlerDeleteUser() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		userId, err := cfg.CheckJwtToken(req)

		if err != nil {
			cfg.HandlerRefresh().ServeHTTP(resp, req)
			return
		}

		usr, err := cfg.Db.GetUserFromId(req.Context(), userId)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte("Could not find user id in database"))
			return
		}

		cfg.Db.DeleteUser(req.Context(), usr.Email)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte("issue removing user id from database"))
			return
		}

		UnsetTokens(&resp)

		resp.Header().Set("HX-Redirect", "/deleteAccountPage")
		resp.WriteHeader(http.StatusNoContent)

	})
}
func (cfg *ApiConfig) HandlerDeletePage() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		htmlComponents, err := MarhalComponent(components.DeleteAccountPage())

		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not render html components, %v", err.Error()))
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return
		}

		resp.Write(htmlComponents.Bytes())
		return

	})
}
