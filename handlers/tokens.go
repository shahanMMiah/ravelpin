package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/internal/auth"
	"github.com/shahanmmiah/ravelpin/internal/database"
)

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

func SetJobCookie(resp *http.ResponseWriter, jobId string) {
	refreshcookie := http.Cookie{
		Name:     SSETOKENCOOKIE,
		Value:    jobId,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(*resp, &refreshcookie)
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
		jb := "homePage"

		SetJobCookie(&resp, jb)
		cfg.UpdatStatus(req, "Status - Hompage")

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
		resp.Header().Set("Content-Type", "text/html")

		resp.WriteHeader(http.StatusOK)
		resp.Write(w.Bytes())

	})
}
