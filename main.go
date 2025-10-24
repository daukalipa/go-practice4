package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// User model
type User struct {
	ID      int     `db:"id"`
	Name    string  `db:"name"`
	Email   string  `db:"email"`
	Balance float64 `db:"balance"`
}

// NewDB opens and configures the DB connection.
func NewDB() (*sqlx.DB, error) {
	connStr := "postgres://user:password@localhost:5430/mydatabase?sslmode=disable"
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func main() {
	db, err := NewDB()
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.Close()

	fmt.Println("Connected to DB. Starting CLI.")
	StartCLI(db)
	fmt.Println("exiting")
}

/*
-------------------------

	CLI
	-------------------------
*/
func StartCLI(db *sqlx.DB) {
	r := bufio.NewReader(os.Stdin)
	fmt.Println("Practice4 CLI — type 'help' for commands")

	for {
		fmt.Print("> ")
		line, _ := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "help":
			printHelp()
		case "exit", "quit":
			fmt.Println("bye")
			return
		case "list":
			users, err := GetAllUsers(db)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			for _, u := range users {
				fmt.Printf("%d: %s <%s> — %.2f\n", u.ID, u.Name, u.Email, u.Balance)
			}
		case "add":
			// add <name> <email> <balance>
			if len(parts) < 4 {
				fmt.Println("usage: add <name> <email> <balance>")
				continue
			}
			bal, err := strconv.ParseFloat(parts[3], 64)
			if err != nil {
				fmt.Println("invalid balance")
				continue
			}
			u := User{Name: parts[1], Email: parts[2], Balance: bal}
			if err := InsertUser(db, u); err != nil {
				fmt.Println("insert error:", err)
			} else {
				fmt.Println("user added")
			}
		case "get":
			// get <id>
			if len(parts) < 2 {
				fmt.Println("usage: get <id>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("invalid id")
				continue
			}
			u, err := GetUserByID(db, id)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			fmt.Printf("%d: %s <%s> — %.2f\n", u.ID, u.Name, u.Email, u.Balance)
		case "transfer":
			// transfer <fromID> <toID> <amount>
			if len(parts) < 4 {
				fmt.Println("usage: transfer <fromID> <toID> <amount>")
				continue
			}
			fromID, err1 := strconv.Atoi(parts[1])
			toID, err2 := strconv.Atoi(parts[2])
			amt, err3 := strconv.ParseFloat(parts[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				fmt.Println("invalid arguments")
				continue
			}
			if err := TransferBalance(db, fromID, toID, amt); err != nil {
				fmt.Println("transfer failed:", err)
			} else {
				fmt.Println("transfer ok")
			}
		default:
			fmt.Println("unknown command — type 'help'")
		}
	}
}

func printHelp() {
	fmt.Println(`Commands:
  help                             Show this help
  list                             List all users
  add <name> <email> <balance>     Add new user
  get <id>                         Show user by id
  transfer <from> <to> <amount>    Transfer money
  exit, quit                       Exit CLI`)
}

/* -------------------------
   DB helpers / CRUD / Tx
   ------------------------- */

// InsertUser inserts a new user (uses NamedExec)
func InsertUser(db *sqlx.DB, user User) error {
	query := `INSERT INTO users (name, email, balance) VALUES (:name, :email, :balance)`
	_, err := db.NamedExec(query, user)
	return err
}

// GetAllUsers returns all users
func GetAllUsers(db *sqlx.DB) ([]User, error) {
	query := `SELECT id, name, email, balance::double precision AS balance FROM users ORDER BY id`
	var users []User
	err := db.Select(&users, query)
	return users, err
}

// GetUserByID returns a single user by id
func GetUserByID(db *sqlx.DB, id int) (User, error) {
	query := `SELECT id, name, email, balance::double precision AS balance FROM users WHERE id=$1`
	var u User
	err := db.Get(&u, query, id)
	return u, err
}

// TransferBalance performs a safe transfer inside a transaction
func TransferBalance(db *sqlx.DB, fromID int, toID int, amount float64) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	// if any error happens, rollback (safe to call even after commit)
	defer func() {
		_ = tx.Rollback()
	}()

	// Lock sender row FOR UPDATE and check balance
	var fromBalance float64
	if err := tx.Get(&fromBalance, `SELECT balance::double precision AS balance FROM users WHERE id=$1 FOR UPDATE`, fromID); err != nil {
		return fmt.Errorf("sender select: %w", err)
	}
	if fromBalance < amount {
		return fmt.Errorf("insufficient funds: have %.2f need %.2f", fromBalance, amount)
	}

	// Lock receiver row FOR UPDATE
	var _to float64
	if err := tx.Get(&_to, `SELECT balance::double precision AS balance FROM users WHERE id=$1 FOR UPDATE`, toID); err != nil {
		return fmt.Errorf("receiver select: %w", err)
	}

	// Update balances
	if _, err := tx.Exec(`UPDATE users SET balance = balance - $1 WHERE id = $2`, amount, fromID); err != nil {
		return fmt.Errorf("debit: %w", err)
	}
	if _, err := tx.Exec(`UPDATE users SET balance = balance + $1 WHERE id = $2`, amount, toID); err != nil {
		return fmt.Errorf("credit: %w", err)
	}

	// commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
