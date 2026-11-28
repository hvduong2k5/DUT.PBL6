package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
)

func customerTraceability(service *app.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if request.URL.RawQuery != "" {
			guestFailure(service, w, domain.E("UNSUPPORTED_QUERY", 400))
			return
		}
		result, err := service.CustomerTraceability(request.Context(), chi.URLParam(request, "id"), principal(request))
		if err != nil {
			guestFailure(service, w, err)
			return
		}
		write(w, 200, result)
	}
}
