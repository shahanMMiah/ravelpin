package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/internal/services"
)

func (cfg *ApiConfig) HandlerUserSearches() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		userId, err := cfg.CheckJwtToken(req)
		if err != nil {
			cfg.HandlerRefresh().ServeHTTP(resp, req)
			return
		}

		userHist, err := cfg.Db.GetUserSearches(req.Context(), userId)
		if err != nil {
			ErrorMsg(&resp, req, http.StatusInternalServerError, "error getting user search history", nil)
		}
		posts := make([]services.UserSearch, 0)

		for _, hist := range userHist {
			post := services.UserSearch{SearchId: hist.ID, Photo: hist.SearchImg}
			posts = append(posts, post)

		}

		htmlComponents := components.UserSearches(posts)
		w, err := MarhalComponent(htmlComponents)

		if err != nil {
			headers := map[string]string{"Content-Type": "text/html"}
			ErrorMsg(&resp, req, http.StatusInternalServerError, fmt.Sprintf("could not render html components, %v", err.Error()), headers)
			return

		}

		resp.WriteHeader(http.StatusOK)
		resp.Header().Set("Content-Type", "text/html")
		resp.Write(w.Bytes())

	})
}

func (cfg *ApiConfig) HandlerUserSearchResult() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		searchId := req.URL.Query().Get("id")
		headers := map[string]string{"Content-Type": "text/html"}

		if searchId == "" {
			ErrorMsg(&resp, req, http.StatusInternalServerError, "error parsing searching id url parameter", headers)
			return
		}

		searchUid, err := uuid.Parse(searchId)
		if err != nil {
			ErrorMsg(&resp, req, http.StatusInternalServerError, fmt.Sprintf("error parsing searching id to uuid %v", searchId), headers)
			return
		}
		results, err := cfg.Db.GetUserSearchResults(req.Context(), searchUid)
		if err != nil {
			ErrorMsg(&resp, req, http.StatusInternalServerError, fmt.Sprintf("no results found for search id %v", searchId), headers)
			return
		}
		posts := make([]services.RavelryPattern, 0)

		for _, post := range results {
			post := services.RavelryPattern{Id: int(post.SearchID.ID()), Name: post.Name, Permalink: post.Permalink, FirstPhoto: services.RavelPhoto{MediumURL: post.ImagePath}}
			posts = append(posts, post)

		}

		htmlComponents := components.SearchResults(posts)
		w, err := MarhalComponent(htmlComponents)

		if err != nil {
			headers := map[string]string{"Content-Type": "text/html"}
			ErrorMsg(&resp, req, http.StatusInternalServerError, fmt.Sprintf("could not render html components, %v", err.Error()), headers)
			return

		}

		resp.WriteHeader(http.StatusOK)
		resp.Header().Set("Content-Type", "text/html")
		resp.Write(w.Bytes())

	})
}
