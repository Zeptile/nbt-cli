package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

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

type exitCoder interface {
	error
	ExitCode() int
}

type cliError struct {
	code int
	err  error
}

func (e *cliError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *cliError) ExitCode() int {
	return e.code
}

func exitError(code int, err error) error {
	return &cliError{code: code, err: err}
}

func exitErrorf(code int, format string, args ...any) error {
	return exitError(code, fmt.Errorf(format, args...))
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "nbt-cli",
		Short:         "Minecraft NBT map editor for block entities",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newMapCmd())

	return root
}

func newMapCmd() *cobra.Command {
	cf := &commonFlags{}
	cmd := &cobra.Command{
		Use:   "map",
		Short: "Manage block entity data within region files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&cf.regionDir, "region-dir", "", "Path to region directory containing r.*.*.mca")
	cmd.PersistentFlags().StringVar(&cf.regionFile, "region-file", "", "Path to single region file .mca")
	cmd.PersistentFlags().IntVar(&cf.x, "x", 0, "Block X coordinate")
	cmd.PersistentFlags().IntVar(&cf.y, "y", 0, "Block Y coordinate")
	cmd.PersistentFlags().IntVar(&cf.z, "z", 0, "Block Z coordinate")

	cmd.AddCommand(
		newMapGetCmd(cf),
		newMapCreateCmd(cf),
		newMapDeleteCmd(cf),
	)

	return cmd
}

func newMapGetCmd(cf *commonFlags) *cobra.Command {
	var printRegion bool

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Inspect the block entity at the given coordinates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMapGet(cf, printRegion)
		},
	}

	cmd.Flags().BoolVar(&printRegion, "print-region", false, "Also print region path to stdout on second line")

	return cmd
}

func newMapCreateCmd(cf *commonFlags) *cobra.Command {
	var (
		id          string
		data        string
		dataFile    string
		printRegion bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create or update a block entity at the given coordinates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMapCreate(cf, id, data, dataFile, printRegion)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Block entity id (e.g. minecraft:chest)")
	cmd.Flags().StringVar(&data, "data", "", "JSON for extra NBT fields")
	cmd.Flags().StringVar(&dataFile, "data-file", "", "Path to JSON file for extra NBT fields")
	cmd.Flags().BoolVar(&printRegion, "print-region", false, "Also print region path to stdout on second line")

	return cmd
}

func newMapDeleteCmd(cf *commonFlags) *cobra.Command {
	var printRegion bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete the block entity at the given coordinates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMapDelete(cf, printRegion)
		},
	}

	cmd.Flags().BoolVar(&printRegion, "print-region", false, "Also print region path to stdout on second line")

	return cmd
}

func runMapGet(cf *commonFlags, printRegion bool) error {
	r, path, err := openRegion(cf)
	if err != nil {
		return exitErrorf(1, "open region: %w", err)
	}
	defer r.Close()

	chunk, _, _, _, _, err := loadChunk(r, cf.x, cf.z)
	if err != nil {
		return exitErrorf(1, "load chunk: %w", err)
	}

	ent, ok := chunkedit.GetBlockEntity(chunk, cf.x, cf.y, cf.z)
	if !ok {
		return exitErrorf(2, "not found at (%d,%d,%d) in %s", cf.x, cf.y, cf.z, path)
	}

	out, err := json.MarshalIndent(ent, "", "  ")
	if err != nil {
		return exitErrorf(1, "encode entity: %w", err)
	}

	fmt.Println(string(out))
	if printRegion {
		fmt.Println("region:", path)
	} else {
		fmt.Fprintln(os.Stderr, "region:", path)
	}

	return nil
}

func runMapCreate(cf *commonFlags, id, data, dataFile string, printRegion bool) error {
	if data == "" && dataFile != "" {
		contents, err := os.ReadFile(dataFile)
		if err != nil {
			return exitErrorf(1, "read data file: %w", err)
		}
		data = string(contents)
	}

	r, path, err := openRegion(cf)
	if err != nil {
		return exitErrorf(1, "open region: %w", err)
	}
	defer r.Close()

	chunk, cx, cz, _, _, err := loadChunk(r, cf.x, cf.z)
	if err != nil {
		return exitErrorf(1, "load chunk: %w", err)
	}

	extra := map[string]any{}
	if err := chunkedit.MergeJSONData(extra, data); err != nil {
		return exitErrorf(1, "merge data: %w", err)
	}

	chunkedit.CreateOrUpdateBlockEntity(chunk, cf.x, cf.y, cf.z, id, extra)

	if err := r.WriteChunkNBT(cx, cz, chunk); err != nil {
		return exitErrorf(1, "write chunk: %w", err)
	}

	fmt.Println("ok")
	if printRegion {
		fmt.Println("region:", path)
	} else {
		fmt.Fprintln(os.Stderr, "region:", path)
	}

	return nil
}

func runMapDelete(cf *commonFlags, printRegion bool) error {
	r, path, err := openRegion(cf)
	if err != nil {
		return exitErrorf(1, "open region: %w", err)
	}
	defer r.Close()

	chunk, cx, cz, _, _, err := loadChunk(r, cf.x, cf.z)
	if err != nil {
		return exitErrorf(1, "load chunk: %w", err)
	}

	if !chunkedit.DeleteBlockEntity(chunk, cf.x, cf.y, cf.z) {
		return exitErrorf(2, "not found at (%d,%d,%d)", cf.x, cf.y, cf.z)
	}

	if err := r.WriteChunkNBT(cx, cz, chunk); err != nil {
		return exitErrorf(1, "write chunk: %w", err)
	}

	fmt.Println("ok")
	if printRegion {
		fmt.Println("region:", path)
	} else {
		fmt.Fprintln(os.Stderr, "region:", path)
	}

	return nil
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

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		if ec, ok := err.(exitCoder); ok {
			if msg := err.Error(); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(ec.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
