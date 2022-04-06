package main

import (
	"io/ioutil"
	"log"
	"mime/multipart"
	"sync"

	"github.com/gin-gonic/gin"
	graphql "github.com/mjarkk/yarql"
)

func main() {
	r := gin.Default()

	graphqlSchema := graphql.NewSchema()
	err := graphqlSchema.Parse(QueryRoot{}, MethodRoot{}, nil)
	if err != nil {
		log.Fatal(err)
	}

	// The GraphQL is not thread safe so we use this lock to prevent race conditions and other errors
	var lock sync.Mutex

	r.Any("/graphql", func(c *gin.Context) {
		var form *multipart.Form

		getForm := func() (*multipart.Form, error) {
			if form != nil {
				return form, nil
			}

			var err error
			form, err = c.MultipartForm()
			return form, err
		}

		lock.Lock()
		defer lock.Unlock()

		res, _ := graphqlSchema.HandleRequest(
			c.Request.Method,
			c.Query,
			func(key string) (string, error) {
				form, err := getForm()
				if err != nil {
					return "", err
				}
				values, ok := form.Value[key]
				if !ok || len(values) == 0 {
					return "", nil
				}
				return values[0], nil
			},
			func() []byte {
				requestBody, _ := ioutil.ReadAll(c.Request.Body)
				return requestBody
			},
			c.ContentType(),
			&graphql.RequestOptions{
				GetFormFile: func(key string) (*multipart.FileHeader, error) {
					form, err := getForm()
					if err != nil {
						return nil, err
					}
					files, ok := form.File[key]
					if !ok || len(files) == 0 {
						return nil, nil
					}
					return files[0], nil
				},
				Tracing: true,
			},
		)

		c.Data(200, "application/json", res)
	})

	r.Run()
}
