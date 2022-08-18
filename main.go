package main

import (
	"fmt"
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

func die(err error, msgs ...string) {
	for _, msg := range msgs {
		fmt.Fprintln(os.Stderr, msg)
	}
	fmt.Fprintln(os.Stderr, "FATAL:", err)
	os.Exit(1)
}

func main() {
	targetRevName := "master"
	sourceRevName := ""
	repositoryPath := "."
	if len(os.Args) > 1 {
		targetRevName = os.Args[1]
	}
	if len(os.Args) > 2 {
		sourceRevName = os.Args[2]
	}
	if len(os.Args) > 3 {
		repositoryPath = os.Args[3]
	}

	r, err := git.PlainOpen(repositoryPath)
	if err != nil {
		die(err, "Unable to open git repo.")
	}

	targetHash, err := r.ResolveRevision(plumbing.Revision(targetRevName))
	if err != nil {
		die(err, "Unable to resolve target.")
	}

	var sourceHash plumbing.Hash
	if sourceRevName != "" {
		sourceHashPtr, err := r.ResolveRevision(plumbing.Revision(sourceRevName))
		if err != nil {
			die(err, "Unable to resolve source.")
		}
		sourceHash = *sourceHashPtr
	} else {
		sourceRef, err := r.Head()
		if err != nil {
			die(err, "Unable to resolve HEAD.")
		}
		sourceHash = sourceRef.Hash()
	}

	source, err := r.CommitObject(sourceHash)
	if err != nil {
		die(err, "Unable to load source.")
	}

	target, err := r.CommitObject(*targetHash)
	if err != nil {
		die(err, "Unable to load target.")
	}

	if isAncestor, err := source.IsAncestor(target); err != nil {
		die(err, "Unable to determine merge status.")
	} else if isAncestor {
		fmt.Println("merged")
		os.Exit(0)
	}

	commits, err := target.MergeBase(source)
	if err != nil {
		die(err, "Unable to find merge base.")
	}

	mergeBases := map[plumbing.Hash]struct{}{}
	for _, c := range commits {
		mergeBases[c.Hash] = struct{}{}
	}

	iter := object.NewCommitIterCTime(source, nil, nil)
	iter.ForEach(func(c *object.Commit) error {
		if _, ok := mergeBases[c.Hash]; ok {
			return storer.ErrStop
		}
		fmt.Println("----------")
		fmt.Println("commit", c.Hash)
		if len(c.ParentHashes) > 1 {
			fmt.Print("merge")
			for _, hash := range c.ParentHashes {
				fmt.Print(" ", shortHash(r, hash.String()))
			}
			fmt.Println()
		}
		fmt.Println()
		fmt.Println(c.Message)
		return nil
	})
}

func shortHash(r *git.Repository, s string) string {
	n := len(s)
	for i := 6; i <= n; i++ {
		hash := s[0:i]
		_, err := r.ResolveRevision(plumbing.Revision(hash))
		if err == nil {
			return hash
		}
	}
	return ""
}
