// Command maturity prints the manufactory's object maturity report: for
// each kind, how much geometry it is made of, the details it can be made
// at, and how many of the numbers it reads are measured and how many
// picked.
//
// usage, from slayer's root:
//
//	maturity [-family NAME] [-numbers] [KIND...]
//
// With no KIND every kind is listed; with some, only those. -family keeps
// the kinds of one family, as the family column names it. -numbers lists,
// under each kind, the names of the numbers it reads.
//
// The report counts; it does not judge. A kind with many parts and no
// measured number is a detailed guess. See
// doc/model/spacecraft/object-maturity.md.
//
// exit 0, or 1 if the report cannot be made, or 2 on a kind or a family
// that is not there or a flag it does not know.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/janearc/slayer/manufacture"
)

// main prints the report and exits with run's status.
func main() { os.Exit(run(os.Args, os.Stdout, os.Stderr, manufacture.MaturityReport)) }

// run is the command: it reads the flags after the command's own name,
// which is the first of args, makes the report with report,
// keeps the rows asked for and prints them to out.
func run(args []string, out, errOut io.Writer,
	report func() ([]manufacture.Maturity, error)) int {
	flags := flag.NewFlagSet("maturity", flag.ContinueOnError)
	flags.SetOutput(errOut)
	family := flags.String("family", "", "keep only the kinds of this family")
	listed := flags.Bool("numbers", false, "list the numbers each kind reads")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	rows, err := report()
	if err != nil {
		fmt.Fprintln(errOut, "maturity:", err)
		return 1
	}
	kept, err := keep(rows, *family, flags.Args())
	if err != nil {
		fmt.Fprintln(errOut, "maturity:", err)
		return 2
	}
	write(out, kept, *listed)
	return 0
}

// keep is the rows of one family, if a family is named, and of the kinds
// named, if any are. A family or a kind the report does not hold is
// refused, with the ones it does hold.
func keep(rows []manufacture.Maturity, family string, kinds []string) ([]manufacture.Maturity, error) {
	var names, families []string
	for _, row := range rows {
		names = append(names, row.Name)
		if !slices.Contains(families, row.Family) {
			families = append(families, row.Family)
		}
	}
	slices.Sort(families)
	if family != "" && !slices.Contains(families, family) {
		return nil, fmt.Errorf("no family %q; there are %s", family,
			strings.Join(families, ", "))
	}
	for _, kind := range kinds {
		if !slices.Contains(names, kind) {
			return nil, fmt.Errorf("no kind %q; there are %s", kind,
				strings.Join(names, ", "))
		}
	}
	var kept []manufacture.Maturity
	for _, row := range rows {
		if (family == "" || row.Family == family) &&
			(len(kinds) == 0 || slices.Contains(kinds, row.Name)) {
			kept = append(kept, row)
		}
	}
	return kept, nil
}

// write prints one line a kind, in columns, and with listed the names of
// its numbers beneath it.
func write(out io.Writer, rows []manufacture.Maturity, listed bool) {
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "kind\tfamily\tparts\tpieces\tmesh faces\tlooks\tlights\tdetails\tsize, metres\tmeasured\tpicked")
	for _, row := range rows {
		fmt.Fprintf(table, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%.1f x %.1f x %.1f\t%d\t%d\n",
			row.Name, row.Family, row.Parts, row.Pieces, row.MeshFaces, row.Looks, row.Lights,
			strings.Join(row.Details, " "), row.Size[0], row.Size[1], row.Size[2],
			len(row.Measured), len(row.Picked))
		if listed {
			for _, name := range row.Measured {
				fmt.Fprintf(table, "\tmeasured: %s\n", name)
			}
			for _, name := range row.Picked {
				fmt.Fprintf(table, "\tpicked: %s\n", name)
			}
		}
	}
	table.Flush()
}
