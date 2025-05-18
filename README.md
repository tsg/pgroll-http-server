# Simple HTTP Server for Xata pgroll

This is a simple HTTP server that allows you to manage Xata pgroll migrations over an HTTP API.

## Usage / tutorial

Export `PG_CONN_STRING` to your Xata / Postgres database. Optionally export `PG_SCHEMA` as well, defaults to `public`.

Run the server:

```bash
export PG_CONN_STRING="..."
export PG_SCHEMA="..."
go run .
```

Then you first initialize pgroll:

```bash
curl -X POST http://localhost:8080/init
```

And you can start a migration that creates a table like this:

```bash
curl -XPOST localhost:8080/start-migration -d'{                                                   
  "name": "0001_create_foo_table",
  "operations": [{
    "create_table": {"name": "foo", "columns": [{"name": "id", "type": "serial", "pk": true}]}
  }]
}'
```

After a migration is started, you can either complete it:

```bash
curl -X POST http://localhost:8080/complete-migration
```

Or rollback it:

```bash
curl -X POST http://localhost:8080/rollback-migration
```

You can also start and complete a migration in one go. Here is an example that adds a new column to the `foo` table:

```bash
curl -XPOST localhost:8080/start-and-complete-migration -d '{
  "name": "0003_add_column_bar", 
  "operations": [
    {
      "add_column": {
        "table": "foo",
        "column": {"name": "bar", "type": "text", "nullable": true}
      }
    }
  ]
}'
```

You can find the documentation for the supported operations in the [pgroll docs](https://pgroll.com/docs/latest/operations/add_column).
