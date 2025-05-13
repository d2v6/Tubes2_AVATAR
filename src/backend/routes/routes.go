package routes

import (
	elementsController "backend/controllers"
	"backend/websocket"
	"encoding/json"
	"log"
	"net/http"
	"sort"
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
        AllowCredentials: true,
    }))

    controller, err := elementsController.NewElementController("data/elements.json")
    if err != nil {
        log.Fatalf("failed to initialize controller: %v", err)
    }

    r.Route("/api", func(r chi.Router) {
        r.Get("/tiers", handleGetAllElementsTiers)
        r.Get("/elements/{name}", handleGetElementByName(controller))
    })

    r.Get("/ws/tree", websocket.HandleTreeWebSocket(controller))

    fs := http.FileServer(http.Dir("frontend/dist"))
    r.Handle("/*", fs)

    return r
}

func handleGetElementByName(controller *elementsController.ElementController) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        name := chi.URLParam(r, "name")
        if name == "" {
            http.Error(w, "element name is required", http.StatusBadRequest)
            return
        }

        element, err := controller.GetElementByName(name)
        if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(element)
    }
}

func handleGetAllElementsTiers(w http.ResponseWriter, r *http.Request) {
    controller, err := elementsController.NewElementController("data/elements.json")
    if err != nil {
        http.Error(w, "failed to initialize controller", http.StatusInternalServerError)
        return
    }

    tierGroups, err := controller.GetAllElementsTiers()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "tiers": extractTierNumbers(tierGroups),
        "elements": tierGroups,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func extractTierNumbers(tierGroups map[string][]string) []int {
    tiers := make([]int, 0, len(tierGroups))
    for tierStr := range tierGroups {
        tier, _ := strconv.Atoi(tierStr)
        tiers = append(tiers, tier)
    }
    sort.Ints(tiers)
    return tiers
}