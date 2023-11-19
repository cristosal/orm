# dbx
[![Go Reference](https://pkg.go.dev/badge/github.com/cristosal/dbx.svg)](https://pkg.go.dev/github.com/cristosal/pgxx)

A library that faciliatates sql queries and struct mappings with sql.DB

## Features
- Unified support for singular connections, pools and transactions via `DB` interface
- Inserts that assign id field using returning clause
- Support for generics
- Database column mapping via `db` tags
- Support for pagination
- ReadOnly fields
- Support for embeded structs

## Installation

`go get -u github.com/cristosal/dbx`

## Documentation

View godoc documentation here

https://pkg.go.dev/github.com/cristosal/dbx

## Usage

Define the struct which will map to your postgres table.
- If tags are omited, the fields are mapped to columns matching their snake_cased values.
- If TableName() method is not implemented then the snake_cased struct name is used.

```go
type User struct {
    ID          dbx.ID `db:"id"`
    Username    string  `db:"username"`
    Password    string  `db:"password"`
    Confirmed   bool    `db:"confirmed_at"`
}

func (*User) TableName() string {
    return "users"
}
```

### Insert
Inserts a row into table. Insert ID is automatically assigned to struct.

```go
u := User{
    Username: "admin",
    Password: "changeme",
}

err := dbx.Insert(db, &u)
```


### One
Collect one row. Takes `sql` argument which is placed after the select statement.

```go
var u User

err := dbx.One(db, &u, "WHERE id = $1", 1)
```
This executes the following sql query:

```sql
SELECT id, username, password FROM users WHERE id = $1
```

### First

Same as `dbx.One` but without an `sql` argument. Returns first row found from table.

```go
var u User

err := dbx.First(db, &u)
```


### Update

Updates an entity by it's `id` field. The following will change the username from admin to superuser.

```go
var u User

err := dbx.One(db, &u, "WHERE username = $1", "admin")

u.Username = "superuser"

err = dbx.Update(db, &u)
```

### Many
Returns all rows which satisfy the query. Takes an `sql` argument which is placed after `select`.

```go
var users []User

err := dbx.Many(db, &users, "WHERE confirmed = TRUE")
```

### Full Example
Here is a simple example of insert, update, and find one.  
**Please Note: Error checking has been omitted for brevity**

```go
package main

import (
    "os"
    "fmt"
    "context"

    "github.com/cristosal/dbx"
    "github.com/jackc/pgx/v5"
)

type Person struct {
    ID      dbx.ID `db:"id"`
    Name    string  `db:"name"`
    Age     int     `db:"age"`
}

// TableName tells dbx which table to use for the given struct
// if not implemented dbx will use the snake-cased version of the struct name ie) person
func (p *Person) TableName() string {
    return "person"
}

func main() {
    conn, _ := pgx.Connect(context.Background(), os.Getenv("CONNECTION_STRING"))

    p := Person{Name: "John Doe", Age: 29}

    _ = dbx.Insert(conn, &p)

    // p.ID is now set to autgenerated id
    fmt.Printf("successfully added person with id %d\n", p.ID)
    p.Age++

    _ = dbx.Update(conn, &p)

    var found Person
    _ = dbx.One(conn, &found, "where name = $1", "John Doe")

    fmt.Printf("%s is %d years old\n", found.Name, found.Age) // John Doe is 30 years old 
}

```
