# pgxx

A library that faciliatates sql queries and struct mappings with pgx

## Features
- Unified support for singular connections, pools and transactions via `pgxx.DB` interface
- Inserts that assign id field using returning clause
- Support for generics
- Database column mapping via `db` tags
- Support for pagination
- ReadOnly fields
- Support for embeded structs

## Installation

`go get -u github.com/cristosal/pgxx`

## Documentation

View godoc documentation here

https://pkg.go.dev/github.com/cristosal/pgxx

## Usage

Here is a simple example of insert, update, and find one.  **Error checking has been omitted for brevity**

```go
package main

import (
    "os"
    "fmt"
    "context"

    "github.com/cristosal/pgxx"
    "github.com/jackc/pgx/v5"
)

type Person struct {
    ID      pgxx.ID `db:"id"`
    Name    string  `db:"name"`
    Age     int     `db:"age"`
}

// TableName tells pgxx which table to use for the given struct
// if not implemented pgxx will use the snake-cased version of the struct name ie) person
func (p *Person) TableName() string {
    return "person"
}

func main() {
    conn, _ := pgx.Connect(context.Background(), os.Getenv("CONNECTION_STRING"))

    p := Person{Name: "John Doe", Age: 29}

    _ = pgxx.Insert(conn, &p)

    // p.ID is now set to autgenerated id
    fmt.Printf("successfully added person with id %d\n", p.ID)
    p.Age++

    _ = pgxx.Update(conn, &p)

    var found Person
    _ = pgxx.One(conn, &found, "where name = $1", "John Doe")

    fmt.Printf("%s is %d years old\n", found.Name, found.Age) // John Doe is 30 years old 
}

```
