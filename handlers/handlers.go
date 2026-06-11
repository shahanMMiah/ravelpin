package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/internal/auth"

	"github.com/shahanmmiah/ravelpin/internal/logging"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
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

// UTILS

func MiddleWareServeFile(file string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		http.ServeFile(resp, req, file)
	})
}

func CheckIfImage(file string) (bool, error) {
	resp, err := http.Head(file)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("status is: %v", resp.Status)
	}

	contentType := resp.Header.Get("content-type")
	if !strings.Contains(contentType, "image") {
		return false, fmt.Errorf("content type not image: %v", contentType)

	}

	return true, nil
}

func MiddleWareModelPreload(model *recoginition.ImageClassifier, mdl func(string, *recoginition.ImageClassifier) ([]string, error)) func(string) ([]string, error) {
	return func(s string) ([]string, error) {
		return mdl(s, model)
	}
}

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

// SERVER HANDLERS

func (cfg *ApiConfig) SetupServer(clasifyModel *recoginition.ClassifyModels) {

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
