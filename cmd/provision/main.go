package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func main() {
	instance := flag.String("instance", "agency", "the instance segment every table is named under")
	apply := flag.Bool("apply", false, "run the statements; without it, print what would run")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fail(fmt.Errorf("DATABASE_URL is not set"))
	}

	s, err := mwanachamaagency.SpecFor(*instance)
	if err != nil {
		fail(err)
	}

	if !*apply {
		for _, stmt := range s.DDL("postgres") {
			fmt.Println(stmt + ";")
		}
		return
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fail(err)
	}

	before := tables(db, *instance)
	if err := mwanachamaagency.Provision(db, s); err != nil {
		fail(err)
	}
	after := tables(db, *instance)

	fmt.Printf("before (%d):\n", len(before))
	for _, t := range before {
		fmt.Println("  " + t)
	}
	fmt.Printf("after (%d):\n", len(after))
	for _, t := range after {
		mark := " "
		if !holds(before, t) {
			mark = "+"
		}
		fmt.Printf("  %s %s\n", mark, t)
	}
}

func tables(db *gorm.DB, instance string) []string {
	var names []string
	if err := db.Raw(
		`select table_name from information_schema.tables where table_schema = 'public' and table_name like ?`,
		instance+"%").Scan(&names).Error; err != nil {
		fail(err)
	}
	sort.Strings(names)
	return names
}

func holds(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "provision: "+strings.TrimSpace(err.Error()))
	os.Exit(1)
}
