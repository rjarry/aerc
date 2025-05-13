package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emersion/go-message/mail"
)

func TestTemplates_DifferentNamesFormats(t *testing.T) {
	type testCase struct {
		address mail.Address
		name    string
	}

	cases := []testCase{
		{address: mail.Address{Name: "", Address: "john@doe.com"}, name: "john"},
		{address: mail.Address{Name: "", Address: "bill.john.doe@doe.com"}, name: "bill.john.doe"},
		{address: mail.Address{Name: "John", Address: "john@doe.com"}, name: "John"},
		{address: mail.Address{Name: "John Doe", Address: "john@doe.com"}, name: "John Doe"},
		{address: mail.Address{Name: "Bill John Doe", Address: "john@doe.com"}, name: "Bill John Doe"},
		{address: mail.Address{Name: "Doe, John", Address: "john@doe.com"}, name: "John Doe"},
		{address: mail.Address{Name: "Doe, Bill John", Address: "john@doe.com"}, name: "Bill John Doe"},
		{address: mail.Address{Name: "Schröder, Gerhard", Address: "s@g.de"}, name: "Gerhard Schröder"},
		{address: mail.Address{Name: "Buhl-Freiherr von und zu Guttenberg, Karl-Theodor Maria Nikolaus Johann Jacob Philipp Franz Joseph Sylvester", Address: "long@email.com"}, name: "Karl-Theodor Maria Nikolaus Johann Jacob Philipp Franz Joseph Sylvester Buhl-Freiherr von und zu Guttenberg"},
		{address: mail.Address{Name: "Dr. Őz-Szűcs Villő, MD, PhD, MBA (Üllői úti Klinika, Budapest, Hungary)", Address: "a@b.com"}, name: "Dr. Őz-Szűcs Villő, MD, PhD, MBA (Üllői úti Klinika, Budapest, Hungary)"},
		{address: mail.Address{Name: "International Important Conference, 2023", Address: "a@b.com"}, name: "2023 International Important Conference"},
		{address: mail.Address{Name: "A. B.C. Muscat", Address: "a@b.com"}, name: "A. B.C. Muscat"},
		{address: mail.Address{Name: "Wertram, te, K.W.", Address: "a@b.com"}, name: "Wertram, te, K.W."},
		{address: mail.Address{Name: "Harvard, John, Dr. CDC/MIT/SYSOPSYS", Address: "a@b.com"}, name: "Harvard, John, Dr. CDC/MIT/SYSOPSYS"},
	}

	for _, c := range cases {
		names := names([]*mail.Address{&c.address})
		assert.Len(t, names, 1)
		assert.Equal(t, c.name, names[0])
	}
}

func TestTemplates_DifferentFirstnamesFormats(t *testing.T) {
	type testCase struct {
		address   mail.Address
		firstname string
	}

	cases := []testCase{
		{address: mail.Address{Name: "", Address: "john@doe.com"}, firstname: "john"},
		{address: mail.Address{Name: "", Address: "bill.john.doe@doe.com"}, firstname: "bill"},
		{address: mail.Address{Name: "John", Address: "john@doe.com"}, firstname: "John"},
		{address: mail.Address{Name: "John Doe", Address: "john@doe.com"}, firstname: "John"},
		{address: mail.Address{Name: "Bill John Doe", Address: "john@doe.com"}, firstname: "Bill"},
		{address: mail.Address{Name: "Doe, John", Address: "john@doe.com"}, firstname: "John"},
		{address: mail.Address{Name: "Schröder, Gerhard", Address: "s@g.de"}, firstname: "Gerhard"},
		{address: mail.Address{Name: "Buhl-Freiherr von und zu Guttenberg, Karl-Theodor Maria Nikolaus Johann Jacob Philipp Franz Joseph Sylvester", Address: "long@email.com"}, firstname: "Karl-Theodor"},
		{address: mail.Address{Name: "Dr. Őz-Szűcs Villő, MD, PhD, MBA (Üllői úti Klinika, Budapest, Hungary)", Address: "a@b.com"}, firstname: "Dr."},
		{address: mail.Address{Name: "International Important Conference, 2023", Address: "a@b.com"}, firstname: "2023"},
		{address: mail.Address{Name: "A. B.C. Muscat", Address: "a@b.com"}, firstname: "A."},
		{address: mail.Address{Name: "Wertram, te, K.W.", Address: "a@b.com"}, firstname: "Wertram"},
		{address: mail.Address{Name: "Harvard, John, Dr. CDC/MIT/SYSOPSYS", Address: "a@b.com"}, firstname: "Harvard"},
	}

	for _, c := range cases {
		names := firstnames([]*mail.Address{&c.address})
		assert.Len(t, names, 1)
		assert.Equal(t, c.firstname, names[0])
	}
}

func TestTemplates_InternalRearrangeNamesWithComma(t *testing.T) {
	type testCase struct {
		source string
		res    string
	}

	cases := []testCase{
		{source: "John.Doe", res: "John.Doe"},
		{source: "John Doe", res: "John Doe"},
		{source: "John Bill Doe", res: "John Bill Doe"},
		{source: "Doe, John Bill", res: "John Bill Doe"},
		{source: "Doe, John-Bill", res: "John-Bill Doe"},
		{source: "Doe John, Bill", res: "Bill Doe John"},
		{source: "Schröder, Gerhard", res: "Gerhard Schröder"},
		// check that we properly trim spaces
		{source: " John Doe", res: "John Doe"},
		{source: "   Doe John,   Bill", res: "Bill Doe John"},
		// do not touch names with more than one comma
		{source: "One, Two, Three", res: "One, Two, Three"},
		{source: "One, Two, Three, Four", res: "One, Two, Three, Four"},
	}

	for _, c := range cases {
		res := rearrangeNameWithComma(c.source)
		assert.Equal(t, c.res, res)
	}
}

func TestTemplates_DifferentInitialsFormats(t *testing.T) {
	type testCase struct {
		address  mail.Address
		initials string
	}

	cases := []testCase{
		{address: mail.Address{Name: "", Address: "john@doe.com"}, initials: "j"},
		{address: mail.Address{Name: "", Address: "bill.john.doe@doe.com"}, initials: "b"},
		{address: mail.Address{Name: "John", Address: "john@doe.com"}, initials: "J"},
		{address: mail.Address{Name: "John Doe", Address: "john@doe.com"}, initials: "JD"},
		{address: mail.Address{Name: "Bill John Doe", Address: "john@doe.com"}, initials: "BJD"},
		{address: mail.Address{Name: "Doe, John", Address: "john@doe.com"}, initials: "JD"},
		{address: mail.Address{Name: "Doe, John Bill", Address: "john@doe.com"}, initials: "JBD"},
		{address: mail.Address{Name: "Schröder, Gerhard", Address: "s@g.de"}, initials: "GS"},
		{address: mail.Address{Name: "Buhl-Freiherr von und zu Guttenberg, Karl-Theodor Maria Nikolaus Johann Jacob Philipp Franz Joseph Sylvester", Address: "long@email.com"}, initials: "KMNJJPFJSBvuzG"},
		{address: mail.Address{Name: "Dr. Őz-Szűcs Villő, MD, PhD, MBA (Üllői úti Klinika, Budapest, Hungary)", Address: "a@b.com"}, initials: "DŐVMPM(úKBH"},
		{address: mail.Address{Name: "International Important Conference, 2023", Address: "a@b.com"}, initials: "2IIC"},
		{address: mail.Address{Name: "A. B.C. Muscat", Address: "a@b.com"}, initials: "ABM"},
		{address: mail.Address{Name: "Wertram, te, K.W.", Address: "a@b.com"}, initials: "WtK"},
		{address: mail.Address{Name: "Harvard, John, Dr. CDC/MIT/SYSOPSYS", Address: "a@b.com"}, initials: "HJDC"},
	}

	for _, c := range cases {
		intls := initials([]*mail.Address{&c.address})
		assert.Len(t, intls, 1)
		assert.Equal(t, c.initials, intls[0])
	}
}

func TestTemplates_Head(t *testing.T) {
	type testCase struct {
		head   uint
		input  string
		output string
	}
	cases := []testCase{
		{head: 3, input: "abcde", output: "abc"},
		{head: 10, input: "abcde", output: "abcde"},
	}

	for _, c := range cases {
		out := head(c.head, c.input)
		assert.Equal(t, c.output, out)
	}
}

func TestTemplates_Tail(t *testing.T) {
	type testCase struct {
		tail   uint
		input  string
		output string
	}
	cases := []testCase{
		{tail: 2, input: "abcde", output: "de"},
		{tail: 8, input: "abcde", output: "abcde"},
	}

	for _, c := range cases {
		out := tail(c.tail, c.input)
		assert.Equal(t, c.output, out)
	}
}
