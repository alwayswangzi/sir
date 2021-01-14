package sir

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Router struct {
	mux *http.ServeMux
}

func New() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (r *Router) ListenAndServe(addr string) {
	if err := http.ListenAndServe(addr, r.mux); err != nil {
		panic(err)
	}
}

func (r *Router) Handler(pattern string, f func(c *Ctx)) {
	r.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		f(newCtx(w, r))
	})
}

func (r *Router) Static(path string) {
	prefix := path[:len(path)-1]
	dir := path[1 : len(path)-1]
	r.mux.Handle(path, http.StripPrefix(prefix, http.FileServer(http.Dir(dir))))
}

func (r *Router) Template(path, file string) {
	t, err := template.ParseFiles(file)
	if err != nil {
		log.Fatal(err)
	}
	r.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if err := t.Execute(w, nil); err != nil {
			log.Fatal(err)
		}
	})
}

type Ctx struct {
	w http.ResponseWriter
	r *http.Request
}

func newCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{w, r}
}

func (c *Ctx) URI() string {
	return c.r.RequestURI
}

func (c *Ctx) GetQuery() url.Values {
	return c.r.URL.Query()
}

func (c *Ctx) Bind(v interface{}) error {
	if c.r.Header.Get("Content-Type") != "application/json" {
		return errors.New("content-type not support")
	}
	if body, err := ioutil.ReadAll(c.r.Body); err != nil {
		return err
	} else if len(body) != 0 {
		if err = json.Unmarshal(body, v); err != nil {
			return err
		}
	}
	return nil
}

func (c *Ctx) Execute(t *template.Template, data interface{}) error {
	return t.Execute(c.w, data)
}

func (c *Ctx) Raw(body []byte) {
	if _, err := c.w.Write(body); err != nil {
		LogError(err)
	}
}

func (c *Ctx) Json(resp map[string]interface{}) {
	if body, err := json.Marshal(resp); err != nil {
		LogError(err)
	} else if _, err = c.w.Write(body); err != nil {
		LogError(err)
	}
}

func (c *Ctx) Success(data ...interface{}) {
	resp := map[string]interface{}{
		"code": 0,
		"msg":  "ok",
	}
	if len(data) == 1 {
		resp["data"] = data[0]
	} else {
		resp["data"] = data
	}
	c.Json(resp)
}

func (c *Ctx) Fail(errs ...error) {
	msg := []string{"error!"}
	for _, err := range errs {
		LogError(err, 3)
		msg = append(msg, err.Error())
	}
	resp := map[string]interface{}{
		"code": -1,
		"msg":  strings.Join(msg, " "),
	}
	c.Json(resp)
}

func (c *Ctx) ErrorRequest(statusCode int, err error) {
	c.w.WriteHeader(statusCode)
	if err != nil {
		_, _ = c.w.Write([]byte(err.Error()))
	}
}

func (c *Ctx) BadRequest(err ...error) {
	if len(err) == 1 {
		c.ErrorRequest(http.StatusBadRequest, err[0])
	} else {
		c.ErrorRequest(http.StatusBadRequest, nil)
	}
}

func (c *Ctx) NotFoundRequest(err ...error) {
	if len(err) == 1 {
		c.ErrorRequest(http.StatusNotFound, err[0])
	} else {
		c.ErrorRequest(http.StatusNotFound, nil)
	}
}

func (c *Ctx) FormFile(key string, maxSize int64) ([]byte, *multipart.FileHeader, error) {
	c.r.Body = http.MaxBytesReader(c.w, c.r.Body, maxSize)
	if err := c.r.ParseMultipartForm(maxSize); err != nil {
		return nil, nil, fmt.Errorf("parse form failed! %v, max size: %dKB", err, maxSize/1000)
	}
	file, fileHeader, err := c.r.FormFile(key)
	if err != nil {
		return nil, nil, fmt.Errorf("get formfile failed, key:%s, %v", key, err)
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("read file failed! %v, file header: %v", err, fileHeader)
	}
	return bytes, fileHeader, nil
}

func (c *Ctx) Download(filename string, file []byte) {
	c.w.Header().Add("Content-type", http.DetectContentType(file))
	c.w.Header().Add("Content-Length", strconv.FormatInt(int64(len(file)), 10))
	c.w.Header().Add("content-disposition", "attachment; filename=\""+filename+"\"")

	if size, err := c.w.Write(file); err != nil {
		LogError(fmt.Errorf("write file data err:%v, size:%d", err, size), 3)
	}
}

func (c *Ctx) Image(img []byte) {
	c.w.Header().Set("Accept-Ranges", "bytes")
	c.w.Header().Set("Content-Type", http.DetectContentType(img))

	if size, err := c.w.Write(img); err != nil {
		LogError(fmt.Errorf("write image err:%v, size:%d", err, size), 3)
	}
}
