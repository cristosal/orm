# orm

[![Go Reference](https://pkg.go.dev/badge/github.com/cristosal/dbx.svg)](https://pkg.go.dev/github.com/cristosal/pgxx)

## Features

- Struct to column mapping via `db` tags
- Database schema management via migrations
## Installation

`go get -u github.com/crisatosal/orm`

## Documentation

View go doc documentation here

https://pkg.go.dev/github.com/cristosal/orm

## Usage

First let's create a struct that represents our table. 

```go
type User struct {
    ID       int64
    Name     string
    Username string
    Password string
    Active   bool
}
```

By default, the snake_cased name of the struct will be used as the table name. In this case our `User` struct will map to the `user` table in our database.
Likewise, all fields in the struct will map to a snake_cased column names.

Let's say we want this struct to map instead to a `users` table instead of a `user` table. To do this we define the `TableName` method on the struct.

```go
func (User) TableName() string {
    return "users"
}
```

Now open up an `sql.DB`. For our example we are using the `pgx` driver, but you can use whatever driver you want.

```go
db, err := sql.Open("pgx", os.Getenv("CONNECTION_STRING"))
```

Now we can pass this `db` around to our orm functions.
### Add

To insert our user into the database we call the `Add` function. This function will automatically set the ID of our user to the value generated by the database.

```go
u := User{
    Name:     "John Doe",
    Username: "john_doe@example.com",
    Password: "changeme",
    Active:   true,
}

err := orm.Add(db, &u)
// TODO: handle error

fmt.Printf("added user with id=%d", u.ID)
```

### Get

Now that we have added our user, let's retrieve it from the database.  First declare the type that will be scanned to.

```go
var u User
```

Now we get the user from the database. Note that whatever is contained in the SQL string is placed after the `SELECT` statement.

```go
err := orm.Get(db, &u, "WHERE id = $1", 1)
```

This executes the following SQL query:

```sql
SELECT id, name, username, password FROM users WHERE id = $1
```

### List

Lets take a look at all our active users in our database. 

Like `Get` we must pass a pointer to scan to, but this time we will pass a slice so that we can read all results from the result set.

```go
var users []User

err := orm.List(db, &users, "WHERE active = TRUE")
```

>If we wanted to list all users without needing any additional SQL, we could just pass an empty string or use the `orm.All` function

### Update

Let's assume John Doe wants to change their name. To do this we use the `Update`  function.  Like the `Get` function the SQL string allows you to customize the query. Here we are updating by id.

```go
// Change the name
u.Name = "Bob Smith"

err := orm.Update(db, &u, "WHERE id = $1", u.ID)
```

Since this is a common SQL query we can also use the `UpdateByID` variant.

```go
err := orm.UpdateByID(db, &u)
```

### Remove

Our user decided they want to delete their account. Let's remove them from the database. The function is similar to update

```go
err := orm.Remove(db, &u, "WHERE id = $1", u.ID)
```

Remove also has a `ByID` variant

```go
err := orm.RemoveByID(db, &u)
```


## Migrations

In order to change your database schema over time you can use the migration features built in to `orm`.

Initialize migration tables

```go
err := orm.CreateMigrationTable(db)
```

You can  change the name of the table or the schema that is used for  migrations if you wish.

```go
orm.SetMigrationTable("my_migrations") // defaults to _migrations
orm.SetSchema("my_schema")             // defaults to public
```

The core functionality is encompassed in the following methods

```go
// AddMigration adds a migration to the database and executes it.
// No error is returned if migration was already executed
func AddMigration(db.DB, migration *Migration) error

// RemoveMigration executes the down migration removes the migration from db
func RemoveMigration(db.DB) error
```

### Add Migration

Example of adding/ a migration

```go
orm.AddMigration(db, &orm.Migration{
	Name:        "Create Users Table",
	Description: "Add Users Table with username and password fields",
	Up: `CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		password TEXT NOT NULL
	)`,
	Down: "DROP TABLE users"
})
```

### Remove Migration

The most recent migration can then be reversed by calling the Remove method

```go
orm.RemoveMigration(db)
```

