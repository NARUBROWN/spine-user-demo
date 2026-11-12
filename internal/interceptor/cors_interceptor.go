package interceptor

import "github.com/NARUBROWN/spine/interceptor/cors"

func NewCORSInterceptor() *cors.CORSInterceptor {
	return cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	})
}
