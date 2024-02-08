//go:build matr
// +build matr

package main

import (
	"context"
	"flag"
	"fmt"
	iofs "io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matr-builder/matr/matr"
	"github.com/pkg/errors"
	"github.com/quesurifn/ics-calendar-tidbyt-server/pkg/sliceutil"
)

// dirNotExists returns true if the dir does not exist and false otherwise
func dirNotExists(dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return true
		}
	}
	return false
}

// Build will build the requested binary
func Build(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	var platform = fs.String("p", "linux", "platform")
	fs.Parse(args)

	if len(args) < 1 {
		// loop through cmd dirctory to get all build paths
		files, err := ioutil.ReadDir("./cmd")
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			if f.IsDir() {
				args = append(args, f.Name())
			}
		}
	}

	var wg sync.WaitGroup

	for _, b := range args {
		wg.Add(1)
		go func(b string) {
			startTime := time.Now()
			defer func() {
				fmt.Println("Finished Building:", b, time.Now().Sub(startTime))
				wg.Done()
			}()
			fmt.Println("Building:", b)
			cmd := matr.Sh(`GOOS=%s CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -tags timetzdata -o build/%s ./cmd/%s`, *platform, b, b)
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}

		}(b)
	}
	wg.Wait()

	return nil
}

// ServerlessBuild will build the requested sls binary
func Serverless(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("serverless", flag.ExitOnError)
	var platform = fs.String("p", "linux", "platform")
	var arch = fs.String("a", "amd64", "architecture")
	fs.Parse(args)

	// if no args are provided just go over the `./sls` directory as root
	if len(args) < 1 {
		rootDir := "./sls"
		if dirNotExists(rootDir) {
			log.Fatalf("root dir '%s' was not found\n", rootDir)
		}

		if err := filepath.WalkDir(rootDir, func(path string, entry iofs.DirEntry, _ error) error {
			// Look for all `main.go` files inside the `rootDir`, build them and store the `bootstrap`
			// binary in the same path where the `main.go` was found.
			if !entry.IsDir() && strings.HasSuffix(path, "main.go") {
				args = append(args, path)
			}
			return nil

		}); err != nil {
			log.Fatal(err)
		}
	}

	// let's process each binary found
	var wg sync.WaitGroup

	shellCmd := strings.Join([]string{
		"GOARCH=%s GOOS=%s CGO_ENABLED=0",           // go build tags
		`go build -ldflags '-extldflags "-static"'`, // build extra information
		"-tags lambda.norpc",                        // disable Lambda RPC
		"-o %s/bootstrap %s",                        // output the binary in the same directory where the `main.go` is
	}, " ")

	for _, file := range args {
		wg.Add(1)
		go func(file string) {
			dirname := filepath.Dir(file)
			startTime := time.Now()
			defer func() {
				fmt.Printf("+ Built: %s\n  output: %s/bootstrap\n  in: %s\n", file, dirname, time.Now().Sub(startTime))
				wg.Done()
			}()

			fmt.Printf("- Building: %s\n  with command template: %s\n", file, fmt.Sprintf(shellCmd, *arch, *platform, dirname, file))

			cmd := matr.Sh(shellCmd, *arch, *platform, dirname, file)
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				fmt.Println()
			}

		}(file)
	}
	wg.Wait()

	return nil
}

// Lint run linters against codebase
func Lint(ctx context.Context, args []string) error {
	if len(args) < 1 {
		args = []string{"go", "proto"}
	}

	if sliceutil.Contains(args, "go") {
		fmt.Println("Running GolangCI-Lint...")
		if err := matr.Sh(`go run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...`).Run(); err != nil {
			fmt.Println("Proto Gen")
			return errors.Wrap(err, "[GO-LINT ERRORS]")
		}
	}

	if sliceutil.Contains(args, "proto") {
		fmt.Println("Running Buf Lint...")
		if err := matr.Sh(`go run github.com/bufbuild/buf/cmd/buf lint`).Run(); err != nil {
			return errors.Wrap(err, "[BUF-LINT ERRORS]")
		}
	}

	return nil
}

// Docker will build the docker images for all services
func Docker(ctx context.Context, args []string) error {
	dockerfile := "."
	imgName := "bookgrpc"
	if len(args) > 0 {
		dockerfile = args[0] + ".Dockerfile"
		imgName = args[0]
	}
	fmt.Println("Building Docker Image")
	if err := matr.Sh(`docker build %s -t allergan-data-labs/%s:latest`, dockerfile, imgName).Run(); err != nil {
		return errors.Wrap(err, "[DOCKER ERROR]")
	}

	return nil
}

// Proto generates all the protobuf based artifacts
func Proto(ctx context.Context, args []string) error {
	if err := matr.Sh(`go run github.com/bufbuild/buf/cmd/buf generate`).Run(); err != nil {
		return errors.Wrap(err, "[PROTO-GEN ERROR]")
	}

	return nil
}

// Gql generates the graphql model after updating the graphqls file
func Gql(ctx context.Context, args []string) error {
	c := matr.Sh(`go run github.com/99designs/gqlgen generate --config gqlgen.yml`)
	c.Dir = "gql"
	if err := c.Run(); err != nil {
		fmt.Println("Gql Gen")
		return errors.Wrap(err, "[GQL-GEN ERROR]")
	}

	return nil
}

// Generate regenerates the Protobuf schema and GraphQL model
func Generate(ctx context.Context, args []string) error {
	err := matr.Deps(ctx, Proto, Gql)
	if err != nil {
		return errors.Wrap(err, "[GENERATE ERROR]")
	}

	return nil
}

// Run will run the requested program (bkgrpc, bkgql or bkctl) if left empty bkgrpc and bkgql will run
func Run(ctx context.Context, args []string) error {
	if len(args) == 0 {

		go func() {
			fmt.Println(matr.Sh(`go run ./cmd/server`).Run())
		}()
		// keep things going
		select {}
	}

	if err := matr.Sh(`go run ./cmd/` + strings.Join(args, " ")).Run(); err != nil {
		return err
	}

	return nil
}

func DBMigrate(ctx context.Context, args []string) error {
	if err := matr.Sh(fmt.Sprintf(`go run ./cmd/dbmigrate %s`, strings.Join(args, " "))).Run(); err != nil {
		return errors.Wrap(err, "[DBMIGRATE ERROR]")
	}

	return nil
}

// LoadTestGRPC will run the load test supplied
func LoadTestGRPC(ctx context.Context, args []string) error {
	c := matr.Sh(`go run load_testing/grpc/grpc.go ` + strings.Join(args, " "))
	if err := c.Run(); err != nil {
		return errors.Wrap(err, "[LOAD-TEST ERROR]")
	}

	return nil
}
