package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"text/template"

	"github.com/paulmach/orb/maptile"
	"github.com/xdbsoft/mongosm/mongodb"

	"github.com/dustin/go-heatmap/schemes"
	"github.com/xdbsoft/mongosm/heatmap"
)

func main() {

	db, err := mongodb.New(context.Background(), os.Getenv("MONGODB_URI"), os.Getenv("MONGODB_DATABASE"))
	if err != nil {
		log.Fatal(err)
	}

	s := server{
		db: db,
	}

	log.Print("starting server...")
	http.HandleFunc("/tiles/", s.tilesHandler)
	http.HandleFunc("/", pageHandler)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

var urlRegex = regexp.MustCompile(`\A/.*/(\d+)/(\d+)/(\d+)\.(png|jpeg)\z`)

func (s *server) tilesHandler(w http.ResponseWriter, r *http.Request) {
	//Split URL
	m := urlRegex.FindStringSubmatch(r.URL.Path)
	if m == nil || len(m) != 5 {
		log.Print("Invalid URL: ", r.URL.Path, m)
		http.NotFound(w, r)
		return
	}

	//Decode level, x and y
	level, err := strconv.Atoi(m[1])
	if err != nil {
		log.Print("Error decoding level: ", m[1])
		http.NotFound(w, r)
		return
	}

	x, err := strconv.Atoi(m[2])
	if err != nil {
		log.Print("Error decoding x: ", m[2])
		http.NotFound(w, r)
		return
	}
	y, err := strconv.Atoi(m[3])
	if err != nil {
		log.Print("Error decoding y: ", m[3])
		http.NotFound(w, r)
		return
	}

	tile := maptile.New(uint32(x), uint32(y), maptile.Zoom(level))

	tileData, err := s.getRaw(r.Context(), tile)

	if err != nil {
		log.Print("Error: ", level, x, y, err)
		http.NotFound(w, r)
		return
	}

	w.Write(tileData)
}

type server struct {
	db *mongodb.Client
}

func (s *server) getRaw(ctx context.Context, tile maptile.Tile) ([]byte, error) {
	features, err := s.db.FindInBBox(ctx, "nodes", tile.Bound(16./256.))
	if err != nil {
		log.Print("Error: ", tile, err)
		return nil, err
	}
	log.Println("Features in", tile, ":", len(features))

	if len(features) == 0 {
		return nil, fmt.Errorf("Empty tile")
	}

	// 256 is 2^8, thus projecting 8 levels further than tile should give us the pixel
	tLeftTop := maptile.At(tile.Bound().LeftTop(), tile.Z+8)

	points := []image.Point{}
	for _, f := range features {
		t := maptile.At(f.Point(), tile.Z+8)

		points = append(points, image.Point{
			X: int(t.X) - int(tLeftTop.X),
			Y: 256 - (int(t.Y) - int(tLeftTop.Y)),
		})
	}

	dotRadius := 16
	imageSize := image.Rect(0, 0, 256, 256)
	limits := imageSize.Inset(-dotRadius)

	imgf := heatmap.Heatmap(limits.Add(image.Pt(dotRadius, dotRadius)), points, limits, dotRadius*2, 128, schemes.PBJ)

	img := imgf.SubImage(imageSize.Add(image.Pt(dotRadius, dotRadius)))

	b := bytes.Buffer{}
	if err := png.Encode(&b, img); err != nil {
		log.Fatal(err)
	}

	return b.Bytes(), nil
}

type Page struct {
	Title string
}

//go:embed template/*
var f embed.FS

func pageHandler(w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(template.ParseFS(f, "template/layout.html"))

	page := Page{
		Title: "",
	}

	if err := tmpl.Execute(w, page); err != nil {
		log.Print(err)
	}
}
