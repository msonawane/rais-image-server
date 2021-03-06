package main

import (
	"iiif"
	"log"
	"net/http"
	"net/url"
	"os"
	"version"

	"github.com/BurntSushi/toml"
	"github.com/hashicorp/golang-lru"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var tilePath string
var infoCache *lru.Cache
var tileCache *lru.TwoQueueCache

const defaultAddress = ":12415"
const defaultInfoCacheLen = 10000

func main() {
	// Defaults
	viper.SetDefault("Address", defaultAddress)
	viper.SetDefault("InfoCacheLen", defaultInfoCacheLen)

	// Allow all configuration to be in environment variables
	viper.SetEnvPrefix("RAIS")
	viper.AutomaticEnv()

	// Config file options
	viper.SetConfigName("rais")
	viper.AddConfigPath("/etc")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	// CLI flags
	pflag.String("iiif-url", "", `Base URL for serving IIIF requests, e.g., "http://example.com:8888/images/iiif"`)
	viper.BindPFlag("IIIFURL", pflag.CommandLine.Lookup("iiif-url"))
	pflag.String("address", defaultAddress, "http service address")
	viper.BindPFlag("Address", pflag.CommandLine.Lookup("address"))
	pflag.String("tile-path", "", "Base path for images")
	viper.BindPFlag("TilePath", pflag.CommandLine.Lookup("tile-path"))
	pflag.Int("iiif-info-cache-size", defaultInfoCacheLen, "Maximum cached image info entries (IIIF only)")
	viper.BindPFlag("InfoCacheLen", pflag.CommandLine.Lookup("iiif-info-cache-size"))
	pflag.String("capabilities-file", "", "TOML file describing capabilities, rather than everything RAIS supports")
	viper.BindPFlag("CapabilitiesFile", pflag.CommandLine.Lookup("capabilities-file"))

	pflag.Parse()

	// Make sure required values exist
	if !viper.IsSet("TilePath") {
		log.Println("ERROR: --tile-path is required")
		pflag.Usage()
		os.Exit(1)
	}

	// Pull all values we need for all cases
	tilePath = viper.GetString("TilePath")
	address := viper.GetString("Address")

	// Handle IIIF data only if we have a IIIF URL
	if viper.IsSet("IIIFURL") {
		iiifURL := viper.GetString("IIIFURL")
		iiifBase, err := url.Parse(iiifURL)
		if err != nil || iiifBase.Scheme == "" || iiifBase.Host == "" || iiifBase.Path == "" {
			log.Fatalf("Invalid IIIF URL (%s) specified: %s", iiifURL, err)
		}

		icl := viper.GetInt("InfoCacheLen")
		if icl > 0 {
			infoCache, err = lru.New(icl)
			if err != nil {
				log.Fatalf("Unable to start info cache: %s", err)
			}
		}

		tcl := viper.GetInt("TileCacheLen")
		if tcl > 0 {
			log.Printf("Creating a tile cache to hold up to %d tiles", tcl)
			tileCache, err = lru.New2Q(tcl)
			if err != nil {
				log.Fatalf("Unable to start info cache: %s", err)
			}
		}

		log.Printf("IIIF enabled at %s\n", iiifBase.String())
		ih := NewIIIFHandler(iiifBase, tilePath)

		if viper.IsSet("CapabilitiesFile") {
			filename := viper.GetString("CapabilitiesFile")
			ih.FeatureSet = &iiif.FeatureSet{}
			_, err := toml.DecodeFile(filename, &ih.FeatureSet)
			if err != nil {
				log.Fatalf("Invalid file or formatting in capabilities file '%s'", filename)
			}
			log.Printf("Setting IIIF capabilities from file '%s'", filename)
		}

		http.HandleFunc(ih.Base.Path+"/", ih.Route)
	}

	http.HandleFunc("/images/tiles/", TileHandler)
	http.HandleFunc("/images/resize/", ResizeHandler)
	http.HandleFunc("/version", VersionHandler)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Error starting listener: %s", err)
	}
}

// VersionHandler spits out the raw version string to the browser
func VersionHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(version.Version))
}
