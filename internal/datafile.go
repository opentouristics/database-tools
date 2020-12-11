package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Datafile represents structure of data.json file.
type Datafile struct {
	Meta     Meta      `json:"meta"`
	Sections []Section `json:"sections"`
	Tracks   []Track   `json:"tracks"`
	Stories  []Story   `json:"stories"`
	Dayrooms []Dayroom `json:"dayrooms"`
}

// Meta represents JSON object in the beginning of data.json file.
type Meta struct {
	RegionID     string    `json:"region_id"`
	RegionName   string    `json:"region_name"`
	GeneratedAt  time.Time `json:"generated_at"`
	Contributors []string  `json:"contributors"`
	Featured     []string  `json:"featured"`
	Sources      []struct {
		Name       string `json:"name"`
		WebsiteURL string `json:"website_url"`
	} `json:"sources"`
}

// Parse parses datafile's metadata and assigns it to meta
// struct pointed to by m.
func (m *Meta) Parse(lang string) error {
	m.GeneratedAt = time.Now().Round(time.Minute).UTC()

	name, err := readFromFile(lang + "/name.txt")
	if err != nil {
		return err
	}
	m.RegionName = string(name)

	data, err := readFromFile("data.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	return nil
}

// Parseable is everything that can be parsed from the database filesystem.
type Parseable interface {
	Parse(lang string) error
}

// Section represents places of similiar type and associated metadata.
type Section struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Icon      string  `json:"icon"`
	BgImage   string  `json:"background_image"`
	QuickInfo string  `json:"quick_info"`
	Places    []Place `json:"places"`
}

// Parse parses section data from its directory and assigns
// it to section pointed to by s. It must be used directly
// in the scetions's directory. It recursively parses places.
func (s *Section) Parse(lang string) error {
	name, err := readFromFile("content/" + lang + "/name.txt")
	if err != nil {
		return err
	}
	s.Name = string(name)

	quickInfo, err := readFromFile("content/" + lang + "/quick_info.txt")
	if err != nil {
		return err
	}
	s.QuickInfo = string(quickInfo)

	data, err := readFromFile("data.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, s)

	// Parse places.
	places := make([]Place, 0, 50)
	placesWalker := func(path string, info os.FileInfo, err error) error {
		level := strings.Count(path, "/")
		if level != 1 {
			return nil
		}

		os.Chdir(path)

		var place Place
		err = place.Parse(lang)
		if err != nil {
			return errors.Wrapf(err, "error parsing %s", path)
		}
		os.Chdir("../..")

		places = append(places, place)
		return nil
	}

	err = filepath.Walk("places", placesWalker)

	s.Places = places

	fmt.Println("generate: section", s.ID)
	for _, place := range s.Places {
		fmt.Println("generate: place", place.ID)
		for i, path := range place.imagesPaths {
			fmt.Printf("%d: %s\n", i, path)
		}
	}

	return err
}

// Place represents single place in real world.
type Place struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Section     string   `json:"section"`
	Icon        string   `json:"icon"`
	QuickInfo   string   `json:"quick_info"`
	Overview    string   `json:"overview"`
	Lat         float32  `json:"lat"`
	Lng         float32  `json:"lng"`
	WebsiteURL  string   `json:"website_url"`
	FacebookURL string   `json:"facebook_url"`
	Headers     []string `json:"headers"`
	Content     []string `json:"content"`
	Images      []string `json:"images"`
	imagesPaths []string
}

// Parse parses place data from its directory and assigns
// it to track pointed to by p. It must be used directly
// in the place's directory.
func (p *Place) Parse(lang string) error {
	name, err := readFromFile("content/" + lang + "/name.txt")
	if err != nil {
		return err
	}
	p.Name = string(name)

	quickInfo, err := readFromFile("content/" + lang + "/quick_info.txt")
	if err != nil {
		return err
	}
	p.QuickInfo = string(quickInfo)

	overview, err := readFromFile("content/" + lang + "/overview.txt")
	if err != nil {
		return err
	}
	p.Overview = string(overview)

	i := 0
	for {
		textFileName := "text_" + fmt.Sprint(i)
		textFile, err := os.Open("content/" + lang + "/" + textFileName)
		if os.IsNotExist(err) {
			break
		}

		header, content, err := readTextualData(textFile)
		if err != nil {
			return err
		}

		p.Headers = append(p.Headers, header)
		p.Content = append(p.Content, content)
	}

	data, err := readFromFile("data.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, p)
	if err != nil {
		return err
	}

	p.makeImagesPaths(Compressed)

	return err
}

func (p *Place) makeImagesPaths(quality Quality) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var qualityDir string
	if quality == Compressed {
		qualityDir = "compressed"
	} else if quality == Original {
		qualityDir = "original"
	}

	// s.Images were set when the story was parsed from its JSON.
	for _, image := range p.Images {
		absPath := filepath.Join(cwd, "images/", qualityDir, "/"+image+".webp")

		if _, err := os.Stat(absPath); err != nil {
			fmt.Printf("generate: error: image at %s does not exist!\n", absPath)
		}

		p.imagesPaths = append(p.imagesPaths, absPath)
	}

	return nil
}

// ImagesPaths returns paths of all images of place p. They are
// specific to your machine!
func (p *Place) ImagesPaths() []string {
	return p.imagesPaths
}

// Track represents a bike trail or some other "long" geographical object.
type Track struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	QuickInfo string   `json:"quick_info"`
	Overview  string   `json:"overview"`
	Images    []string `json:"images"`
	Coords    []struct {
		Lat float32 `json:"lat"`
		Lng float32 `json:"lng"`
	} `json:"coords"`
}

// Parse parses track data from its directory and assigns
// it to track pointed to by t. It must be used directly
// in the track's directory, usually by using os.Chdir().
func (t *Track) Parse(lang string) error {
	name, err := readFromFile(lang + "/name.txt")
	if err != nil {
		return err
	}
	t.Name = string(name)

	overview, err := readFromFile(lang + "/overview.txt")
	if err != nil {
		return err
	}
	t.Overview = string(overview)

	quickInfo, err := readFromFile(lang + "/quick_info.txt")
	if err != nil {
		return err
	}
	t.QuickInfo = string(quickInfo)

	data, err := readFromFile("data.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, t)

	return err
}

// Story represents a longer piece of text about a particular topic.
type Story struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	MarkdownFile string   `json:"markdown_filename"`
	Images       []string `json:"images"`
	imagesPaths  []string
}

// Parse parses story data from its directory and assigns
// it to story pointed to by s. It must be used directly
// in the tracks's directory.
func (s *Story) Parse(lang string) error {
	name, err := readFromFile(lang + "/name.txt")
	if err != nil {
		return err
	}
	s.Name = string(name)

	data, err := readFromFile("data.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, s)
	if err != nil {
		return err
	}

	s.makeImagesPaths(Compressed)

	return err
}

func (s *Story) makeImagesPaths(quality Quality) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var qualityDir string
	if quality == Compressed {
		qualityDir = "compressed"
	} else if quality == Original {
		qualityDir = "original"
	}

	// s.Images were set when the story was parsed from its JSON.
	for _, image := range s.Images {
		absPath := filepath.Join(cwd, "images/", qualityDir, "/"+image+".webp")
		s.imagesPaths = append(s.imagesPaths, absPath)
	}

	return nil
}

// ImagesPaths returns paths of all images of story s. They are
// specific to your machine!
func (s *Story) ImagesPaths() []string {
	return s.imagesPaths
}

// Dayroom represents a place run by local community.
type Dayroom struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Section   string   `json:"section"`
	Name      string   `json:"name"`
	QuickInfo string   `json:"quick_info"`
	Overview  string   `json:"overview"`
	Images    []string `json:"images"`
	Lat       float32  `json:"lat"`
	Lng       float32  `json:"lng"`
	Leader    string   `json:"leader"`
}

// Parse parses dayroom data from its directory and assigns
// it to dayroom pointer to by d. It must be used directly
// in the dayroom's directory, usually by using os.Chdir().
func (dayroom *Dayroom) Parse(lang string) error {
	name, err := readFromFile("content/" + lang + "/name.txt")
	if err != nil {
		return err
	}
	dayroom.Name = string(name)

	overview, err := readFromFile("content/" + lang + "/overview.txt")
	if err != nil {
		return err
	}
	dayroom.Overview = string(overview)

	quickInfo, err := readFromFile("content/" + lang + "/quick_info.txt")
	if err != nil {
		return err
	}
	dayroom.QuickInfo = string(quickInfo)

	data, err := readFromFile("data.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &dayroom)
	if err != nil {
		return err
	}

	return nil
}

// Quality represents the quality of the image.
type Quality int

const (
	// Compressed quality is most often used.
	Compressed = iota + 1
	// Original quality represents full, uncompressed image.
	Original
)

// // Image represents a single image with some metadata *in the datafile*.
// type Image struct {
// 	Name    string
// 	Ext     string
// 	quality ImageQuality
// }
