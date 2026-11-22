package bistro

import (
	"fmt"
	"os"
	"strings"

	"github.com/loilo-inc/bistro/pkg/dpkg"
	"github.com/loilo-inc/bistro/pkg/rootfs"
)

type Baker interface {
	CreateRootFS(input *BakeInput) error
}

type baker struct {
	DI
}

var _ Baker = &baker{}

func NewBaker(di DI) Baker {
	return &baker{DI: di}
}

type BakeInput struct {
	Out      rootfs.Path
	DpkgList []*dpkg.Package
	Files    []rootfs.Path
	Excludes rootfs.PathSet
	Users    []User
	Groups   []Group
}

func (b *baker) CreateRootFS(input *BakeInput) (err error) {
	// Distroless metadata filenames use the unqualified package name.
	// Reject collisions before creating an archive that would lose metadata.
	names := make(map[string]string, len(input.DpkgList))
	for _, pkg := range input.DpkgList {
		if previous, ok := names[pkg.Name()]; ok {
			return fmt.Errorf("package metadata name conflict: %s and %s use %s", previous, pkg.NameWithArch(), pkg.Name())
		}
		names[pkg.Name()] = pkg.NameWithArch()
	}
	outDir := input.Out.Dir()
	tmpFile, err := os.CreateTemp(outDir.Abs(), "rootfs")
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	defer func() {
		if err != nil {
			os.Remove(tmpFile.Name())
		}
	}()
	tmpPath := rootfs.MustPath(tmpFile.Name())
	input.Excludes.Add(input.Out, tmpPath)
	archiver := rootfs.NewArchiver(b.FS, tmpFile)
	archiver.SetExcludes(input.Excludes)

	for _, pkg := range input.DpkgList {
		if err := b.writePackageMetadata(archiver, pkg, input.Excludes); err != nil {
			return err
		}
	}
	for _, file := range input.Files {
		if err := archiver.CopyRecursive(file); err != nil {
			return err
		}
	}
	if err := assertUserGroups(input.Users, input.Groups); err != nil {
		return err
	}
	if err := b.writeUsers(archiver, input.Users); err != nil {
		return err
	}
	if err := b.writeGroups(archiver, input.Groups); err != nil {
		return err
	}
	if err := archiver.Close(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath.Abs(), input.Out.Abs()); err != nil {
		return err
	}
	return nil
}

// https://github.com/GoogleContainerTools/distroless/blob/main/PACKAGE_METADATA.md
// directory to store dpkg status used by distroless
const DpkgStatusDir = "/var/lib/dpkg/status.d"

func (b *baker) writePackageMetadata(
	rootFS rootfs.Archiver,
	pkg *dpkg.Package,
	excludes rootfs.PathSet,
) error {
	if err := rootFS.Add(rootfs.MustPath(DpkgStatusDir).Join(pkg.Name()), []byte(pkg.String()), 0644); err != nil {
		return err
	}
	md5sums, err := pkg.Md5Sums(b.FS)
	if err != nil {
		return err
	}
	if err = rootFS.Add(rootfs.MustPath(DpkgStatusDir).Join(pkg.Name()+".md5sums"), []byte(md5sums.String()), 0644); err != nil {
		return err
	}
	files, err := pkg.ListFiles(b.FS)
	if err != nil {
		return err
	}

	for _, srcPath := range files {
		if skip, err := excludes.Match(b.FS, srcPath); err != nil {
			return err
		} else if skip {
			continue
		}
		if err := rootFS.Copy(srcPath); os.IsNotExist(err) {
			b.Logger.Error("File does not exist: %s", srcPath)
			continue
		} else if err != nil {
			return err
		}
	}
	return nil
}

func assertUserGroups(users []User, groups []Group) error {
	gids := make(map[int]struct{})
	for _, group := range groups {
		if group.Gid == 0 {
			return fmt.Errorf("[baker] Group with GID 0 is not allowed")
		}
		gids[group.Gid] = struct{}{}
	}
	for _, user := range users {
		if user.Gid == 0 || user.Uid == 0 {
			return fmt.Errorf("[baker] User %s has GID 0 or UID 0 which is not allowed", user.Name)
		}
		if _, ok := gids[user.Gid]; !ok {
			return fmt.Errorf("User %s has GID %d which is not defined in groups", user.Name, user.Gid)
		}
	}
	return nil
}

func (b *baker) writeUsers(archiver rootfs.Archiver, users []User) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "root:x:0:0:root:/root:/sbin/nologin\n")
	for _, user := range users {
		fmt.Fprintf(&sb, "%s:x:%d:%d::/home/%s:/sbin/nologin\n", user.Name, user.Uid, user.Gid, user.Name)
	}
	return archiver.Add(rootfs.MustPath("/etc/passwd"), []byte(sb.String()), 0644)
}

func (b *baker) writeGroups(archiver rootfs.Archiver, groups []Group) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "root:x:0:\n")
	for _, group := range groups {
		fmt.Fprintf(&sb, "%s:x:%d:\n", group.Name, group.Gid)
	}
	return archiver.Add(rootfs.MustPath("/etc/group"), []byte(sb.String()), 0644)
}
