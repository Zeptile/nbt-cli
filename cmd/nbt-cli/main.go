package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"nbt-cli/internal/anvil"
	"nbt-cli/internal/chunkedit"
	"nbt-cli/internal/coords"
)

type commonFlags struct {
	regionDir  string
	regionFile string
	x          int
	y          int
	z          int
}

func parseCommon(fs *flag.FlagSet, cf *commonFlags) {
	fs.StringVar(&cf.regionDir, "region-dir", "", "Path to region directory containing r.*.*.mca")
	fs.StringVar(&cf.regionFile, "region-file", "", "Path to single region file .mca")
	fs.IntVar(&cf.x, "x", 0, "Block X coordinate")
	fs.IntVar(&cf.y, "y", 0, "Block Y coordinate")
	fs.IntVar(&cf.z, "z", 0, "Block Z coordinate")
}

func openRegion(cf *commonFlags) (*anvil.Region, string, error) {
	if cf.regionFile != "" {
		p, err := filepath.Abs(cf.regionFile)
		if err != nil {
			return nil, "", err
		}
		r, err := anvil.OpenRegionFile(p)
		return r, p, err
	}
	if cf.regionDir == "" {
		return nil, "", errors.New("either --region-dir or --region-file must be specified")
	}
	return anvil.OpenRegionForWorldXZ(cf.regionDir, cf.x, cf.z)
}

func loadChunk(r *anvil.Region, x, z int) (map[string]any, int, int, int, int, error) {
	cxAbs, czAbs := coords.WorldToChunkXZ(x, z)
	cx, cz := coords.InRegionChunkIndex(cxAbs, czAbs)
	chunk, err := r.ReadChunkNBT(cx, cz)
	return chunk, cx, cz, cxAbs, czAbs, err
}

func cmdMapGet(args []string) int {
	fs := flag.NewFlagSet("map get", flag.ExitOnError)
	var cf commonFlags
    var printRegion bool
	parseCommon(fs, &cf)
    fs.BoolVar(&printRegion, "print-region", false, "Also print region path to stdout on second line")
	fs.Parse(args)
	r, path, err := openRegion(&cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer r.Close()
	chunk, _, _, _, _, err := loadChunk(r, cf.x, cf.z)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	ent, ok := chunkedit.GetBlockEntity(chunk, cf.x, cf.y, cf.z)
	if !ok {
		fmt.Fprintf(os.Stderr, "not found at (%d,%d,%d) in %s\n", cf.x, cf.y, cf.z, path)
		return 2
	}
	out, _ := json.MarshalIndent(ent, "", "  ")
	fmt.Println(string(out))
	if printRegion {
		fmt.Println("region:", path)
	} else {
		fmt.Fprintln(os.Stderr, "region:", path)
	}
	return 0
}

func cmdMapCreate(args []string) int {
	fs := flag.NewFlagSet("map create", flag.ExitOnError)
	var cf commonFlags
	var id string
	var data string
	var dataFile string
    var printRegion bool
	parseCommon(fs, &cf)
	fs.StringVar(&id, "id", "", "Block entity id (e.g. minecraft:chest)")
	fs.StringVar(&data, "data", "", "JSON for extra NBT fields")
	fs.StringVar(&dataFile, "data-file", "", "Path to JSON file for extra NBT fields")
    fs.BoolVar(&printRegion, "print-region", false, "Also print region path to stdout on second line")
	fs.Parse(args)
	if data == "" && dataFile != "" {
		b, err := ioutil.ReadFile(dataFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		data = string(b)
	}
    r, path, err := openRegion(&cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer r.Close()
	chunk, cx, cz, _, _, err := loadChunk(r, cf.x, cf.z)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	extra := map[string]any{}
	if err := chunkedit.MergeJSONData(extra, data); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	chunkedit.CreateOrUpdateBlockEntity(chunk, cf.x, cf.y, cf.z, id, extra)
	if err := r.WriteChunkNBT(cx, cz, chunk); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Println("ok")
    if printRegion {
        fmt.Println("region:", path)
    } else {
        fmt.Fprintln(os.Stderr, "region:", path)
    }
	return 0
}

func cmdMapDelete(args []string) int {
	fs := flag.NewFlagSet("map delete", flag.ExitOnError)
	var cf commonFlags
    var printRegion bool
    parseCommon(fs, &cf)
    fs.BoolVar(&printRegion, "print-region", false, "Also print region path to stdout on second line")
	fs.Parse(args)
    r, path, err := openRegion(&cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer r.Close()
	chunk, cx, cz, _, _, err := loadChunk(r, cf.x, cf.z)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if !chunkedit.DeleteBlockEntity(chunk, cf.x, cf.y, cf.z) {
		fmt.Fprintf(os.Stderr, "not found at (%d,%d,%d)\n", cf.x, cf.y, cf.z)
		return 2
	}
	if err := r.WriteChunkNBT(cx, cz, chunk); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Println("ok")
    if printRegion {
        fmt.Println("region:", path)
    } else {
        fmt.Fprintln(os.Stderr, "region:", path)
    }
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: nbt-cli map <get|create|delete> [options]\n")
	fmt.Fprintf(os.Stderr, "  Common flags: --region-dir DIR | --region-file FILE --x X --y Y --z Z\n")
    fmt.Fprintf(os.Stderr, "  get flags: --print-region (also print region path to stdout)\n")
    fmt.Fprintf(os.Stderr, "  create flags: --id ID [--data JSON | --data-file PATH] [--print-region]\n")
    fmt.Fprintf(os.Stderr, "  delete flags: [--print-region]\n")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "map":
		if len(os.Args) < 3 {
			usage()
			os.Exit(1)
		}
		sub := os.Args[2]
		var code int
		switch sub {
		case "get":
			code = cmdMapGet(os.Args[3:])
		case "create":
			code = cmdMapCreate(os.Args[3:])
		case "delete":
			code = cmdMapDelete(os.Args[3:])
		default:
			usage()
			code = 1
		}
		os.Exit(code)
	default:
		usage()
		os.Exit(1)
	}
}


