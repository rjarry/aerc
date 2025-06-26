// Author: Jim Idle - jimi@idle.ws / jimi@gatherstars.com
// SPDX-License-Identifier: Apache-2.0

package jwz

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

// Where we are going to store the emails. We know that the test data is of a fair size, so we tell the slice that
// in advance
var Emails = make([]Threadable, 0, 93)

// ls -1 testdata | wc -l
const MessageNumber = 93

// TestMain sets up everything for the other test(s). It essentially parses a largish set of publicly available
// Emails in to a structure that can then be used to perform email threading testing.
func TestMain(m *testing.M) {
	// Parse all the emails in the test directory
	//
	loadEmails()

	// OK, we have a fairly large email set all parsed, so now we can let the real tests run
	//
	os.Exit(m.Run())
}

func loadEmails() {
	_ = filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("cannot process directory/file %s because: %#v", path, err)
			os.Exit(1)
		}

		if d.IsDir() {
			// Skip any directory entries, including the base test data dir
			//
			return nil
		}

		// We only look at files that have an eml extension
		//
		if !strings.HasSuffix(path, ".eml") {
			return nil
		}

		f, e := os.Open(path)
		if e != nil {
			return err
		}
		r, e1 := mail.CreateReader(f)
		_ = f.Close()
		if e1 != nil {
			log.Printf("cannot parse email file error = %v", e1)
			return nil
		}

		// All is good, so let's accumulate the email
		//
		email := NewEmail(r.Header)
		Emails = append(Emails, email)
		return nil
	})
}

// EmailRoot is a structure that implements the ThreadableRoot interface. I have not used ThreadableRoot
// here, but this is what it needs to look like if your input structure is not just a slice of Threadable
type EmailRoot struct {
	// This is some structure of the emails you want to thread, that you know how to traverse
	//
	emails []Threadable

	// You need some sort of position holder, which in this silly example is an index in the struct
	//
	position int
}

// Next sets the internal cursor to the next available Threadable
func (e *EmailRoot) Next() bool {
	e.position = e.position + 1
	if e.position < len(e.emails) {
		return true
	}
	return false
}

// Get returns the Threadable at the current internal cursor position
func (e *EmailRoot) Get() Threadable {
	return e.emails[e.position]
}

// NewThreadableRoot returns a struct instance that can be traversed using the ThreadableRoot interface
func NewThreadableRoot(emails []Threadable) ThreadableRoot {
	tr := &EmailRoot{
		emails:   emails,
		position: -1,
	}
	return tr
}

// Email is structure that implements the Threadable interface - this is what a user of this
// package needs to do.
type Email struct {
	header mail.Header
	next   Threadable
	parent Threadable
	child  Threadable
	dummy  bool
	forID  string
}

func (e *Email) GetNext() Threadable {
	return e.next
}

func (e *Email) GetChild() Threadable {
	return e.child
}

// GetParent the parent Threadable of this node, if any
func (e *Email) GetParent() Threadable {
	return e.parent
}

// GetDate extracts the timestamp from the envelope contained in the supplied Threadable
func (e *Email) GetDate() time.Time {
	// We can have dummies because we are likely to have parsed a set of emails with incomplete threads,
	// where the start of the thread or sub thread was referenced, but we did not get to parse it, at least yet.
	// This means it will be a placeholder as the root for the thread, so we can use the time of the child as the
	// time of this email.
	//
	if e.IsDummy() {
		if e.GetChild() != nil {
			return e.GetChild().GetDate()
		}

		// Protect against having nothing in the children that knows what time it is. So, back to the
		// beginning of time according to Unix
		//
		return time.Unix(0, 0)
	}
	d, err := e.header.Date()
	if err != nil {
		return time.Unix(0, 0)
	}
	return d
}

var idre = regexp.MustCompile("<.*?>")

func (e *Email) MessageThreadID() string {
	if e.dummy {
		return e.forID
	}
	ref := e.header.Get("Message-Id")
	refs := idre.FindAllString(ref, -1)
	if len(refs) > 0 {
		return refs[0]
	}
	return "<bogus-id-in-email>"
}

func (e *Email) MessageThreadReferences() []string {
	if e.dummy {
		return nil
	}

	// This should be a nicely formatted field that has unique IDs enclosed within <>, and each of those should be
	// space separated. However, it isn't as simple as this because all sorts of garbage mail clients have been programmed
	// over the years by people who did not understand what the References field was (I'm looking at you
	// Comcast, for instance). We can get things like:
	//
	//    1) References: Your message of Friday... <actual-ID>      (Some garbage the programmer thought might be useful)
	//    2) References: me@mydomain.com                            (This isn't even a reference, it is the sender's email)
	//    3) References: <ref-1><ref-2><ref-3>                      (Either a pure bug, or they misread the spec)
	//
	// Further to this, we also need to filter out the following:
	//
	//    4) References: <this message-id>                  (The client author places this email as the first in the
	//                                                       reference chain)
	//    5) References: <ref-1><ref-2><ref-1>              A pure bug somewhere in the chain repeats a reference
	//
	// The RFC has now been cleaned up to exactly specify this field, but we have to assume there are still
	// 20 year old email clients out there and cater for them. Especially when we are testing with ancient
	// public email bodies.
	//
	ref := e.header.Get("References")

	// Find all the correctly delimited references, which takes care of 1) and 3)
	//
	rawRefs := idre.FindAllString(ref, -1)

	// Find the message Id, so we can take care of 4)
	//
	m := e.MessageThreadID()

	// Find the From address, so we can deal with 2). Even though ignoring this would be harmless in that we would just
	// think it is an email we never saw, it is wrong not to deal with here. We can avoid the clutter in the database
	// by filtering them out.
	//
	fa, _ := e.header.AddressList("From")

	// Make a set, so we can remove duplicates and deal with 5)
	//
	set := make(map[string]any)

	// This will be our final return set, after de-fucking the references
	//
	refs := make([]string, 0, len(rawRefs))

	// Now we range through the references that the email has given us and make sure that the reference does
	// not run afoul of 2), 4) or 5)
	//
	for _, r := range rawRefs {
		// 2) and 5)
		//
		if _, repeated := set[r]; r != m && !repeated {

			set[r] = nil

			// Technically, From: can have more than one sender (back in the day before email lists
			// got sorted), we will never see this in practice, but, in for a pound, in for a penny
			//
			var found bool = false
			for _, f := range fa {
				if r == "<"+f.Address+">" {
					found = true
					break
				}
			}

			if !found {
				// If we got thorough all of those checks, then Phew! Made it!
				//
				refs = append(refs, r)
			}
		}
	}
	return refs
}

var re = regexp.MustCompile("[Rr][Ee][ \t]*:[ \t]*")

func (e *Email) SimplifiedSubject() string {
	if e.dummy {
		return ""
	}
	subj := e.header.Get("Subject")
	subj = re.ReplaceAllString(subj, "")
	return subj
}

func (e *Email) Subject() string {
	if e.dummy {
		if e.child != nil {
			return e.child.Subject() + " :: node synthesized by https://gatherstars.com/"
		}

		return fmt.Sprintf("Placeholder %s - manufactured by https://gatherstars.com/", e.forID)
	}

	// Add in the date for a bit of extra information
	//
	var sb strings.Builder
	t := e.GetDate()
	sb.WriteString(t.UTC().String())
	sb.WriteString(" : ")
	sb.WriteString(strings.Trim(e.header.Get("Subject"), " "))
	return sb.String()
}

func (e *Email) SubjectIsReply() bool {
	subj := e.header.Get("Subject")
	return re.MatchString(subj)
}

func (e *Email) SetNext(next Threadable) {
	e.next = next
}

func (e *Email) SetChild(kid Threadable) {
	e.child = kid
	if kid != nil {
		kid.SetParent(e)
	}
}

// SetParent allows us to add or change the parent Threadable of this node
func (e *Email) SetParent(parent Threadable) {
	e.parent = parent
}

func (e *Email) MakeDummy(forID string) Threadable {
	return &Email{
		dummy: true,
		forID: forID,
	}
}

func (e *Email) IsDummy() bool {
	return e.dummy
}

func NewEmail(header mail.Header) Threadable {
	e := &Email{
		header: header,
		dummy:  false,
	}
	return e
}

func ExampleThreader_ThreadSlice() {
	// Emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		fmt.Printf("func ThreadSlice() error = %#v", err)
		return
	}

	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(sliceRoot, &nc)
	if nc != MessageNumber {
		fmt.Printf("expected %d emails after threading, but got %d back", MessageNumber, nc)
	} else {
		fmt.Printf("There are %d test emails", nc)
	}
	// Output: There are 93 test emails
}

func TestThreader_ThreadSlice(t1 *testing.T) {
	// Emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		t1.Errorf("func ThreadSlice() error = %#v", err)
	}

	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(sliceRoot, &nc)
	if nc != MessageNumber {
		t1.Errorf("expected %d emails after threading, but got %d back", MessageNumber, nc)
	}
}

func ExampleThreader_ThreadRoot() {
	// Emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	tr := NewThreadableRoot(Emails)
	threader := NewThreader()
	treeRoot, err := threader.ThreadRoot(tr)
	if err != nil {
		fmt.Printf("func ThreadRoot() error = %#v", err)
	}
	if treeRoot == nil {
		fmt.Printf("received no output from the threading algorithm")
	}
	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(treeRoot, &nc)
	if nc != MessageNumber {
		fmt.Printf("expected %d emails after threading, but got %d back", MessageNumber, nc)
	} else {
		fmt.Printf("There are %d test emails", nc)
	}
	// Output: There are 93 test emails
}

func TestThreader_ThreadRoot(t1 *testing.T) {
	// Emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the ThreadableRootInterface to traverse the emails
	//
	tr := NewThreadableRoot(Emails)
	threader := NewThreader()
	treeRoot, err := threader.ThreadRoot(tr)
	if err != nil {
		t1.Errorf("ThreadRoot() error = %#v", err)
	}
	if treeRoot == nil {
		t1.Errorf("received no output from the threading algorithm")
	}
	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(treeRoot, &nc)
	if nc != MessageNumber {
		t1.Errorf("expected %d emails after threading, but got %d back", MessageNumber, nc)
	}
}
