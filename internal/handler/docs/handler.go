package docs

import (
	"net/http"

	"github.com/AmirAbaris/weeto-backend/api"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) UI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}

func (h *Handler) Spec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(api.OpenAPISpec)
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Weeto API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout",
      persistAuthorization: true,
      withCredentials: true,
      requestInterceptor: (req) => {
        req.credentials = "include";
        return req;
      },
      responseInterceptor: (res) => {
        if (res.ok && res.url && /\/auth\/(register|login|refresh)$/.test(res.url)) {
          try {
            const data = JSON.parse(res.text);
            if (data.access_token && window.ui) {
              window.ui.authActions.authorize({
                bearerAuth: {
                  name: "bearerAuth",
                  schema: { type: "http", scheme: "bearer", bearerFormat: "JWT" },
                  value: data.access_token,
                },
              });
            }
          } catch (_) {}
        }
        return res;
      },
    });
  </script>
</body>
</html>`
