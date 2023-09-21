# pgxx

A library that facilitates common sql queries with pgx


## Usage

Here is a simple example of an insert statement

`go get -u github.com/cristosal/pgxx`

```go
package main

import (
    "os"
    "fmt"

    "github.com/cristosal/pgxx"
    "github.com/jackc/pgx/v5"
)

type Person struct {
    Name    string  `db:"name"`
    Age     int     `db:"age"`
}

// TableName tells pgxx which table to use for the given struct
// if not implemented pgxx will use the snake-cased version of the struct name ie) person
func (p *Person) TableName() string {
    return "person"
}

func main() {
    conn, err := pgx.Connect(context.Background(), os.Getenv("CONNECTION_STRING"))
    if err != nil {
        fmt.Println("unable to connect to postgres")
        os.Exit(1)
    }

    p := Person{
        Name: "John Doe",
        Age: 29,
    }

    // inserts into the person table
    if err := pgxx.Insert(conn, &p): err != nil {
        fmt.Printf("error inserting person: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("successfully added person")
}

```
