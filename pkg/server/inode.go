package server

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

// If inode contains a name, we can build a fully qualified tree
type Inode struct {
	Name string
	// The current attributes for this inode
	attrs fuseops.InodeAttributes

	// For directories, entries describe the children of the directory
	// Entries contains a name
	entries []fuseutil.Dirent

	// For files, contents contain the current contents of the file
	contents []byte

	// For symlinks, target is the target of the symlink.
	//
	// INVARIANT: If !isSymlink(), len(target) == 0
	target string

	// extended attributes and values
	xattrs map[string][]byte
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Create a new inode with the supplied attributes, which need not to contain
// time-related information (the inode object will take care of that)
func newInode(name string, attrs fuseops.InodeAttributes) *Inode {
	// Update time info
	now := time.Now()
	attrs.Mtime = now
	attrs.Crtime = now

	// Create the object
	return &Inode{
		Name:   name,
		attrs:  attrs,
		xattrs: make(map[string][]byte),
	}
}

func (in *Inode) CheckInvariants() {
	// INVARIANT: attrs.Mode &^ (os.ModePerm|os.ModeDir|os.ModeSymlink) == 0
	if !(in.attrs.Mode&^(os.ModePerm|os.ModeDir|os.ModeSymlink) == 0) {
		panic(fmt.Sprintf("Unexpected mode: %v", in.attrs.Mode))
	}

	// INVARIANT: !(isDir() && isSymlink())
	if in.isDir() && in.isSymlink() {
		panic(fmt.Sprintf("Unexpected mode: %v", in.attrs.Mode))
	}

	// INVARIANT: attrs.Size == len(contents)
	if in.attrs.Size != uint64(len(in.contents)) {
		panic(fmt.Sprintf(
			"Size mismatch: %d vs. %d",
			in.attrs.Size,
			len(in.contents)))
	}

	// INVARIANT: If !isDir(), len(entries) == 0
	if !in.isDir() && len(in.entries) != 0 {
		panic(fmt.Sprintf("Unexpected entries length: %d", len(in.entries)))
	}

	// INVARIANT: For each i, entries[i].Offset == i+1
	for i, e := range in.entries {
		if !(e.Offset == fuseops.DirOffset(i+1)) {
			panic(fmt.Sprintf("Unexpected offset for index %d: %d", i, e.Offset))
		}
	}

	// INVARIANT: Contains no duplicate names in used entries
	childNames := make(map[string]struct{})
	for _, e := range in.entries {
		if e.Type != fuseutil.DT_Unknown {
			if _, ok := childNames[e.Name]; ok {
				panic(fmt.Sprintf("Duplicate name: %s", e.Name))
			}

			childNames[e.Name] = struct{}{}
		}
	}

	// INVARIANT: If !isFile(), len(contents) == 0
	if !in.isFile() && len(in.contents) != 0 {
		panic(fmt.Sprintf("Unexpected length: %d", len(in.contents)))
	}

	// INVARIANT: If !isSymlink(), len(target) == 0
	if !in.isSymlink() && len(in.target) != 0 {
		panic(fmt.Sprintf("Unexpected target length: %d", len(in.target)))
	}

	return
}

func (in *Inode) isDir() bool {
	return in.attrs.Mode&os.ModeDir != 0
}

func (in *Inode) isSymlink() bool {
	return in.attrs.Mode&os.ModeSymlink != 0
}

func (in *Inode) isFile() bool {
	return !(in.isDir() || in.isSymlink())
}

// Return the index of the child within in.entries, if it exists
//
// RREQUIRES: in.isDir()
func (in *Inode) findChild(name string) (i int, ok bool) {
	if !in.isDir() {
		panic("findChild called on non-directory")
	}

	var e fuseutil.Dirent
	for i, e = range in.entries {
		fmt.Println(e.Name)
		if e.Name == name {
			return i, true
		}
	}

	return 0, false
}

////////////////////////////////////////////////////////////////////////
// Public methods
////////////////////////////////////////////////////////////////////////

// Returns the number of children of the directory
//
// REQUIRES: in.isDir()
func (in *Inode) Len() int {
	var n int
	for _, e := range in.entries {
		if e.Type != fuseutil.DT_Unknown {
			n++
		}
	}

	return n
}

// Find an entry for the given child name and return its inode ID
//
// REQUIRES: in.isDir()
func (in *Inode) LookUpChild(name string) (id fuseops.InodeID, typ fuseutil.DirentType, ok bool) {
	index, ok := in.findChild(name)
	if ok {
		id = in.entries[index].Inode
		typ = in.entries[index].Type
	}

	return id, typ, ok
}

// Add an entry for a child
//
// REQUIRES: in.isDir()
// REQUIRES: dt != fuseutil.DT_Unknown
func (in *Inode) AddChild(id fuseops.InodeID, name string, dt fuseutil.DirentType) {
	var index int

	// Update the modification time
	in.attrs.Mtime = time.Now()

	// No matter where we palce the entry, make sure it has the correct Offset field
	defer func() {
		in.entries[index].Offset = fuseops.DirOffset(index + 1)
	}()

	// Set up the entry
	e := fuseutil.Dirent{
		Inode: id,
		Name:  name,
		Type:  dt,
	}

	// Look for a gap in which we can insert it
	for index = range in.entries {
		if in.entries[index].Type == fuseutil.DT_Unknown {
			in.entries[index] = e
			return
		}
	}

	// Append it to the end
	index = len(in.entries)
	in.entries = append(in.entries, e)
}

// Remove an entry for a child
//
// REQUIRED: in.isDir()
// REQUIRED: An entry for the given name exists
func (in *Inode) RemoveChild(name string) {
	// Update the modification time.
	in.attrs.Mtime = time.Now()

	// Find the entry.
	i, ok := in.findChild(name)
	if !ok {
		panic(fmt.Sprintf("Unknown child: %s", name))
	}

	// Mark it as unused
	in.entries[i] = fuseutil.Dirent{
		Type:   fuseutil.DT_Unknown,
		Offset: fuseops.DirOffset(i + 1),
	}
}

// Server a ReadDir request.
//
// REQUIRES: in.isDir()
func (in *Inode) ReadDir(p []byte, offset int) int {
	if !in.isDir() {
		panic("ReadDir called on non-directory.")
	}

	var n int
	for i := offset; i < len(in.entries); i++ {
		e := in.entries[i]

		// Skip the unused entries
		if e.Type == fuseutil.DT_Unknown {
			continue
		}

		tmp := fuseutil.WriteDirent(p[n:], in.entries[i])
		if tmp == 0 {
			break
		}

		n += tmp
	}

	return n
}

// Read from the files contents. See documentation for ioutil.ReaderAt
//
// REQUIRES: in.isFile()
func (in *Inode) ReadAt(p []byte, off int64) (int, error) {
	if !in.isFile() {
		panic("ReadAt called on non-file.")
	}

	// Ensure the offset is in range
	if off > int64(len(in.contents)) {
		return 0, io.EOF
	}

	// Read what we can
	n := copy(p, in.contents[off:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}

// Write to the files contents. See documentation for iotuil.WriteAt
//
// REQUIRES: in.isFile()
func (in *Inode) WriteAt(p []byte, off int64) (int, error) {
	if !in.isFile() {
		panic("WriteAt called on non-file.")
	}

	// Update the modification time
	in.attrs.Mtime = time.Now()

	// Ensure that the contents slice is long enough
	newLen := int(off) + len(p)
	if len(in.contents) < newLen {
		padding := make([]byte, newLen-len(in.contents))
		in.contents = append(in.contents, padding...)
		in.attrs.Size = uint64(newLen)
	}

	// Copy in the data
	n := copy(in.contents[off:], p)

	// Sanity heck.
	if n != len(p) {
		panic(fmt.Sprintf("Unexpected short copy: %v", n))
	}

	return n, nil
}

// Update attribtues from non-nil parameters
func (in *Inode) SetAttributes(size *uint64, mode *os.FileMode, mtime *time.Time) {
	// Updates the modification time
	in.attrs.Mtime = time.Now()

	// Truncate?
	if size != nil {
		intSize := int(*size)

		// Update contents
		if intSize <= len(in.contents) {
			in.contents = in.contents[:intSize]
		} else {
			padding := make([]byte, intSize-len(in.contents))
			in.contents = append(in.contents, padding...)
		}

		// Update attributes.
		in.attrs.Size = *size
	}

	// Change mode?
	if mode != nil {
		in.attrs.Mode = *mode
	}

	// Change mtime?
	if mtime != nil {
		in.attrs.Mtime = *mtime
	}
}

func (in *Inode) Fallocate(mode uint32, offset uint64, length uint64) error {
	if mode != 0 {
		return fuse.ENOSYS
	}
	newSize := int(offset + length)
	if newSize > len(in.contents) {
		padding := make([]byte, newSize-len(in.contents))
		in.contents = append(in.contents, padding...)
		in.attrs.Size = offset + length
	}
	return nil
}