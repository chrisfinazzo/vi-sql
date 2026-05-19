<div align="center">
  <img src="./assets/logo.svg">
</div>

---

Terminal UI for SQL databases built with passion for the terminal — Browse schemas, run queries, edit rows, inspect query plans, and expose your session to AI tools via a built-in MCP server.

![Query image](./assets/query_autocomplete.png)
![Actions image](./assets/actions.png)
![Styles image](./assets/styles.png)

## Features

- **Multi-tab SQL editor** with syntax highlighting, autocomplete, and query history
- **Table data view** — filter, sort, inline edit, add/delete rows, follow FK's, find references
- **Vim mode** — full Normal / Insert / Visual with operators, motions, and chord sequences
- **Schema browser** — tables, structure, indexes, DDL; create, rename, drop via keybindings
- **EXPLAIN / EXPLAIN ANALYZE** viewer with cost and timing breakdown
- **Import/Export**, multi-format (CSV, JSON, SQL INSERT, Markdown) export and Import
- **MCP server** — lets AI assistants (Claude, Cursor, etc.) to securely send queries to your database
- **Encrypted connections** — passwords stored with AES-256-GCM; optional master password

## Install

```sh
curl -fsSL https://vi-sql.com/install.sh | sh
```

Pin a specific version:

```sh
VI_SQL_VERSION=v0.0.3 curl -fsSL https://vi-sql.com/install.sh | sh
```

Or download a binary directly from the [releases page](https://github.com/kopecmaciej/vi-sql/releases).

### Build from source

Requires Go 1.25+.

```sh
git clone https://github.com/kopecmaciej/vi-sql.git
cd vi-sql
make build
```

### Uninstall

```sh
curl -fsSL https://vi-sql.com/uninstall.sh | sh
```

The script prompts before removing each artifact: the binary, config directory, log file, and any keyring entry.

## Quickstart

Run `vi-sql` and enter your connection details on the welcome screen, or connect directly with a DSN:

```sh
vi-sql --connect postgres://user:pass@localhost/mydb
vi-sql --connect mysql://user:pass@localhost/mydb
vi-sql --connect file:/home/user/data.db
```

Or by saved connection name:

```sh
vi-sql --connection-name mydb
```

Jump straight to a table:

```sh
vi-sql --jump public/users
```

Config and data paths vary by OS. Run `vi-sql --paths` to see the exact locations on your system (config, keybindings, styles, icons, log).

## MCP server

vi-sql ships an HTTP MCP server that AI tools can connect to while the app is running. Enable it from the options page (`o` on the welcome screen) or add to your config:

```yaml
mcp:
  enabled: true
  port: 9741
  allowRead: false  # AI cannot run queries directly
  allowWrite: false # AI cannot modify data
```

With both flags off the AI can only browse your schema and open queries in a new tab — you review and run them yourself.

To allow the AI to execute read-only queries automatically, set `allowRead: true`. If your MCP client prompts you to approve each tool call before it runs (Claude Code does this), every query still requires your explicit acceptance regardless of this flag. Set `allowWrite: true` only if you trust the AI to run `INSERT`/`UPDATE`/`DELETE`/`DROP` without confirmation.

Then point your AI tool at `http://localhost:9741/mcp`. Available tools: `list_schemas`, `list_tables`, `describe_table`, `read_query` (requires `allowRead`), `write_query` (requires `allowWrite`), `get_query_from_tab`.

## License

Apache 2.0
