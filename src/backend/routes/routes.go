package routes

import (
	elementsController "backend/controllers"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func InitRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
	}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/path", handleFindSingleRecipe)
		
		r.Get("/recipes", handleFindMultipleRecipes)
	})

	return r
}

func handleFindSingleRecipe(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	method := r.URL.Query().Get("method")
	
	if target == "" {
		http.Error(w, "target parameter required", http.StatusBadRequest)
		return
	}

	useBFS := method != "dfs"

	controller, err := elementsController.NewElementController("data/elements.json")
	if err != nil {
		http.Error(w, "failed to initialize controller", http.StatusInternalServerError)
		return
	}

	recipes, err := controller.FindNRecipes(target, 1, useBFS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes) 
}

func handleFindMultipleRecipes(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	method := r.URL.Query().Get("method")
	countStr := r.URL.Query().Get("count")
	
	if target == "" {
		http.Error(w, "target parameter required", http.StatusBadRequest)
		return
	}

	useBFS := method != "dfs"
	count := 1
	
	if countStr != "" {
		parsedCount, err := strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "count parameter must be an integer", http.StatusBadRequest)
			return
		}
		count = parsedCount
	}

	controller, err := elementsController.NewElementController("data/elements.json")
	if err != nil {
		http.Error(w, "failed to initialize controller", http.StatusInternalServerError)
		return
	}

	recipes, err := controller.FindNRecipes(target, count, useBFS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}