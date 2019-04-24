package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"google.golang.org/grpc"
)

func main() {
	type School struct {
		Name string `json:"name,omitempty"`
	}

	type loc struct {
		Type   string    `json:"type,omitempty"`
		Coords []float64 `json:"coordinates,omitempty"`
	}

	// If omitempty is not set, then edges with empty values (0 for int/float, "" for string, false
	// for bool) would be created for values not specified explicitly.

	type Person struct {
		Uid      string     `json:"uid,omitempty"`
		Name     string     `json:"name,omitempty"`
		Age      int        `json:"age,omitempty"`
		Dob      *time.Time `json:"dob,omitempty"`
		Married  bool       `json:"married,omitempty"`
		Raw      []byte     `json:"raw_bytes,omitempty"`
		Friends  []Person   `json:"friend,omitempty"`
		Location loc        `json:"loc,omitempty"`
		School   []School   `json:"school,omitempty"`
	}

	conn, err := grpc.Dial("127.0.0.1:9080", grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}
	defer conn.Close()

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)

	op := &api.Operation{}
	op.Schema = `
		name: string @index(exact) .
		age: int .
		married: bool .
		loc: geo .
		dob: datetime .
	`
	//op.DropAll = true

	ctx := context.Background()
	err = dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	dob := time.Date(1980, 01, 01, 23, 0, 0, 0, time.UTC)
	// While setting an object if a struct has a Uid then its properties in the graph are updated
	// else a new node is created.
	// In the example below new nodes for Alice, Bob and Charlie and school are created (since they
	// dont have a Uid).
	p := Person{
		Name:    "Alice",
		Age:     26,
		Married: true,
		Location: loc{
			Type:   "Point",
			Coords: []float64{1.1, 2},
		},
		Dob: &dob,
		Raw: []byte("raw_bytes"),
		Friends: []Person{{
			Name: "Bob",
			Age:  24,
		}, {
			Name: "Charlie",
			Age:  29,
		}},
		School: []School{{
			Name: "Crown Public School",
		}},
	}

	mu := &api.Mutation{
		CommitNow: true,
	}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	assigned, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	// Assigned uids for nodes which were created would be returned in the resp.AssignedUids map.
	variables := map[string]string{"$id": assigned.Uids["blank-0"]}
	q := `query Me($id: string){
		me(func: uid($id)) {
			name
			dob
			age
			loc
			raw_bytes
			married
			friend @filter(eq(name, "Charlie")){
				name
				age
			}
			school {
				name
			}
		}
	}`

	resp, err := dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}

	type Root struct {
		Me []Person `json:"me"`
	}

	var r Root
	err = json.Unmarshal(resp.Json, &r)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("Me: %+v\n", r.Me)
	// R.Me would be same as the person that we set above.

	fmt.Println(string(resp.Json))
	op.DropAll = true

	ctx = context.Background()
	err = dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}
}
