package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	var response *httptest.ResponseRecorder

	// DateTime
	response = makeTestRequest(http.MethodPost, "/date-time", map[string]interface{}{
		"time_at": "2020-01-01T02:02:05+07:00",
	})
	fmt.Printf("%+v\n", response.Body.String()) // [200] {"time_at":"2020-01-01T02:02:05+07:00"}

	response = makeTestRequest(http.MethodPost, "/date-time", map[string]interface{}{
		"time_at": "",
	})
	fmt.Printf("%+v\n", response.Body.String()) // [400] {"error":"must not be empty"}
	response = makeTestRequest(http.MethodPost, "/date-time", map[string]interface{}{
		"time_at": true,
	})
	fmt.Printf("%+v\n", response.Body.String()) // [400] {"error":"not a valid string"}
	response = makeTestRequest(http.MethodPost, "/date-time", map[string]interface{}{
		"time_at": "wrong-format",
	})
	fmt.Printf("%+v\n", response.Body.String()) // [400] {"error":"format must be YYYY-MM-DDTHH:mm:ssZ"}

	// ArrayString
	response = makeTestRequest(http.MethodPost, "/array-string", map[string]interface{}{
		"list": "1,2,3,4",
	})
	fmt.Printf("%+v\n", response.Body.String()) // [200] {"time_at":"2020-01-01T02:02:05+07:00"}

	response = makeTestRequest(http.MethodPost, "/array-string", map[string]interface{}{
		"list": true,
	})
	fmt.Printf("%+v\n", response.Body.String()) // [400] {"error":"must not be empty"}
	response = makeTestRequest(http.MethodPost, "/array-string", map[string]interface{}{
		"list": "",
	})
	fmt.Printf("%+v\n", response.Body.String()) // [400] {"error":"must be a valid string"}
}

var (
	router     *gin.Engine
	routerOnce sync.Once
)

type BadRequestError string

type DateTime struct {
	time time.Time
}

// RFC3339     = "2006-01-02T15:04:05Z07:00"
func (dt DateTime) format() string {
	return time.RFC3339
}

/*
	This receiver function overwrite `fmt.Stringer` which use to print the output
	type Stringer interface {
		String() string
	}
*/
func (dt DateTime) String() string {
	return dt.time.Format(dt.format())
}

/*
	This part implements `json.Marshaler`
	type Marshaler interface {
		MarshalJSON() ([]byte, error)
	}
*/
func (dt DateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(dt.String())
}

/*
	This part implements `json.Unmarshaler`
	type Unmarshaler interface {
		UnmarshalJSON([]byte) error
	}
*/
func (dt *DateTime) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		panic(BadRequestError("not a valid string"))
	}
	if s == "" {
		panic(BadRequestError("must not be empty"))
	}
	t, err := time.Parse(dt.format(), s)
	if err != nil {
		panic(BadRequestError("format must be YYYY-MM-DDTHH:mm:ssZ"))
	}

	dt.time = t

	return nil
}

type ArrayString []string

func (dt ArrayString) separator() string {
	return ","
}

func (dt ArrayString) parse(s string) []string {
	return strings.Split(s, dt.separator())
}

func (dt ArrayString) String() string {
	return strings.Join(dt, dt.separator())
}

func (dt ArrayString) List() []string {
	return dt
}

/*
	This part implements `json.Marshaler`
	type Marshaler interface {
		MarshalJSON() ([]byte, error)
	}
*/
func (dt ArrayString) MarshalJSON() ([]byte, error) {
	return json.Marshal(dt.String())
}

/*
	This part implements `json.Unmarshaler`
	type Unmarshaler interface {
		UnmarshalJSON([]byte) error
	}
*/
func (dt *ArrayString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		panic(BadRequestError("must be a valid string"))
	}
	if s == "" {
		panic(BadRequestError("must not be empty"))
	}

	*dt = dt.parse(s)
	return nil
}

type RequestContentDateTime struct {
	TimeAt DateTime `json:"time_at"`
}

type RequestContentArrayString struct {
	List ArrayString `json:"list"`
}

func getRouter() *gin.Engine {
	routerOnce.Do(func() {
		router = gin.New()

		// panic handler
		router.Use(func(ctx *gin.Context) {
			defer func() {
				if r := recover(); r != nil {
					switch v := r.(type) {
					case BadRequestError:
						ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error": v,
						})
						return
					case error:
						fmt.Println("log error: ", v)
					default:
						ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
							"error": "internal server error",
						})
					}
				}
			}()

			ctx.Next()
		})

		// simple routing
		router.POST("/date-time", func(ctx *gin.Context) {
			var request RequestContentDateTime
			err := ctx.ShouldBind(&request)
			if err != nil {
				panic(err)
			}

			ctx.JSON(http.StatusOK, request)
		})

		router.POST("/array-string", func(ctx *gin.Context) {
			var request RequestContentArrayString
			err := ctx.ShouldBind(&request)
			if err != nil {
				panic(err)
			}

			ctx.JSON(http.StatusOK, request)
		})
	})

	return router
}

func makeTestRequest(method string, url string, body map[string]interface{}) *httptest.ResponseRecorder {
	jsoned, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest(method, url, bytes.NewBuffer(jsoned))
	if err != nil {
		panic(err)
	}
	request.Header.Add("Content-Type", "application/json")

	response := httptest.NewRecorder()

	router := getRouter()

	router.ServeHTTP(response, request)

	return response
}
