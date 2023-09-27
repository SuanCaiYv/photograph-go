package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/h2non/bimg"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Server struct {
	photosPath  string
	previewPath string
}

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT, DELETE, HEAD, PATCH")
	resp.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
	if req.Method == http.MethodOptions {
		return
	}
	path := req.URL.Path
	if strings.HasPrefix(path, "/list") {
		s.list(resp, req)
	} else if strings.HasPrefix(path, "/preview") {
		s.preview(resp, req)
	} else if strings.HasPrefix(path, "/origin") {
		s.origin(resp, req)
	} else {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
}

func (s *Server) list(resp http.ResponseWriter, req *http.Request) {
	dir := "/tmp/photos-" + time.Now().Format("20060102-15:04:05")
	if s.previewPath == "" {
		os.Mkdir(dir, 0755)
		s.previewPath = dir
	}
	filepath.Walk(s.photosPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			_, err = os.Stat(filepath.Join(s.previewPath, info.Name()))
			if os.IsNotExist(err) {
			} else {
				return nil
			}

			buffer, err := bimg.Read(path)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			originImage := bimg.NewImage(buffer)

			size, err := originImage.Size()
			if err != nil {
				fmt.Println(err)
				return nil
			}
			x, y := size.Height, size.Width
			if size.Width > size.Height {
				x, y = y, x
			}
			a, b := x/800, y/600
			if b > a {
				a, b = b, a
			}
			newWidth, newHeight := size.Width/a, size.Height/a
			newImage, _ := bimg.NewImage(buffer).Resize(newWidth, newHeight)
			bimg.Write(filepath.Join(s.previewPath, info.Name()), newImage)
		}

		return nil
	})
	list := make([]struct {
		Src    string `json:"src"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}, 0)
	filepath.Walk(s.previewPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			buffer, err := bimg.Read(path)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			originImage := bimg.NewImage(buffer)

			size, err := originImage.Size()
			if err != nil {
				fmt.Println(err)
				return nil
			}
			list = append(list, struct {
				Src    string `json:"src"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			}{
				Src:    "/preview/" + info.Name(),
				Width:  size.Width,
				Height: size.Height,
			})
		}

		return nil
	})
	bytes, err := json.Marshal(list)
	if err != nil {
		fmt.Println(err)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.Write(bytes)
}

func (s *Server) preview(resp http.ResponseWriter, req *http.Request) {
	filename := strings.TrimPrefix(req.URL.Path, "/preview/")
	http.ServeFile(resp, req, filepath.Join(s.previewPath, filename))
}

func (s *Server) origin(resp http.ResponseWriter, req *http.Request) {
	filename := strings.TrimPrefix(req.URL.Path, "/origin/")
	http.ServeFile(resp, req, filepath.Join(s.photosPath, filename))
}

func main() {
	path := flag.String("path", "", "path to the photos folder.")
	flag.Parse()
	server := &Server{
		photosPath: *path,
	}
	fmt.Println("Listening on port 8190...")
	http.ListenAndServe(":8190", server)
}
