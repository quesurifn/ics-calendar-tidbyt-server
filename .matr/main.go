// +build matr

package main

import (
	"context"
	"os"

	"github.com/matr-builder/matr/matr"
)

func main() {
	// Create new Matr instance
	m := matr.New()
	
	
	//  Build will build the requested binary
	m.Handle(&matr.Task{
		Name: "build",
		Summary: "Build will build the requested binary",
		Doc: `Build will build the requested binary`,
		Handler: Build,
	})
	
	//  ServerlessBuild will build the requested sls binary
	m.Handle(&matr.Task{
		Name: "serverless",
		Summary: "ServerlessBuild will build the requested sls binary",
		Doc: `ServerlessBuild will build the requested sls binary`,
		Handler: Serverless,
	})
	
	//  Lint run linters against codebase
	m.Handle(&matr.Task{
		Name: "lint",
		Summary: "Lint run linters against codebase",
		Doc: `Lint run linters against codebase`,
		Handler: Lint,
	})
	
	//  Docker will build the docker images for all services
	m.Handle(&matr.Task{
		Name: "docker",
		Summary: "Docker will build the docker images for all services",
		Doc: `Docker will build the docker images for all services`,
		Handler: Docker,
	})
	
	//  Proto generates all the protobuf based artifacts
	m.Handle(&matr.Task{
		Name: "proto",
		Summary: "Proto generates all the protobuf based artifacts",
		Doc: `Proto generates all the protobuf based artifacts`,
		Handler: Proto,
	})
	
	//  Gql generates the graphql model after updating the graphqls file
	m.Handle(&matr.Task{
		Name: "gql",
		Summary: "Gql generates the graphql model after updating the graphqls file",
		Doc: `Gql generates the graphql model after updating the graphqls file`,
		Handler: Gql,
	})
	
	//  Generate regenerates the Protobuf schema and GraphQL model
	m.Handle(&matr.Task{
		Name: "generate",
		Summary: "Generate regenerates the Protobuf schema and GraphQL model",
		Doc: `Generate regenerates the Protobuf schema and GraphQL model`,
		Handler: Generate,
	})
	
	//  Run will run the requested program (bkgrpc, bkgql or bkctl) if left empty bkgrpc and bkgql will run
	m.Handle(&matr.Task{
		Name: "run",
		Summary: "Run will run the requested program (bkgrpc, bkgql or bkctl) if left empty bkgrpc and bkgql will run",
		Doc: `Run will run the requested program (bkgrpc, bkgql or bkctl) if left empty bkgrpc and bkgql will run`,
		Handler: Run,
	})
	
	// DBMigrate
	m.Handle(&matr.Task{
		Name: "db-migrate",
		Summary: "",
		Doc: ``,
		Handler: DBMigrate,
	})
	
	//  LoadTestGRPC will run the load test supplied
	m.Handle(&matr.Task{
		Name: "load-test-grpc",
		Summary: "LoadTestGRPC will run the load test supplied",
		Doc: `LoadTestGRPC will run the load test supplied`,
		Handler: LoadTestGRPC,
	})

	// Run Matr
	if err := m.Run(context.Background(), os.Args[1:]...); err != nil {
		os.Stderr.WriteString("ERROR: "+err.Error()+"\n")
	}
}
