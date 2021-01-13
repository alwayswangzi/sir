package sir

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestCtx_Upload(t *testing.T) {
	s := New()
	s.Handler("/upload", func(c *Ctx) {
		filename, bytes, err := c.Upload(1*1000, nil)
		if err != nil {
			c.BadRequest(err)
			return
		}
		c.Success(fmt.Sprint(filename, len(bytes)))
	})
	s.ListenAndServe(":8080")
}

func TestCtx_Download(t *testing.T) {
	s := New()
	s.Handler("/download", func(c *Ctx) {
		file, err := ioutil.ReadFile("sir.go")
		if err != nil {
			c.BadRequest(err)
			return
		}
		err = c.Download("sir.go", file)
		if err != nil {
			c.BadRequest(err)
			return
		}
	})
	s.ListenAndServe(":8080")
}
