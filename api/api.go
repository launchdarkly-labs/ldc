package api

import (
	"context"

	"ldc/api/swagger"
)

var Auth context.Context
var Client *swagger.APIClient

func init() {
	Client = swagger.NewAPIClient(&swagger.Configuration{
		UserAgent: "ldc/0.0.1/go",
	})
}

// TODO
func SetServer(newServer string) {
	Client = swagger.NewAPIClient(&swagger.Configuration{
		BasePath:  newServer,
		UserAgent: "ldc/0.0.1/go",
	})

}

func SetToken(newToken string) {
	Auth = context.WithValue(context.Background(), swagger.ContextAPIKey, swagger.APIKey{
		Key: newToken,
	})
}
