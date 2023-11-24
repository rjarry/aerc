AERC-PATCH(7)

# NAME

aerc-patch - local patch management for *aerc*(1)

# SYNOPSIS

*aerc* provides support for managing local patch sets. In an email-based
software development workflow, there are usually many different locally applied
patch series for testing and reviewing. Managing the local repository can thus
be challenging. With the local patch management system, *aerc* facilitates this
bookkeeping process.

When applying a patch set, *aerc* creates a tag for those commits. With this
tag, the patch set can be tracked and later removed if needed. Patches are
stored in a project data structure which also keeps track of the directory where
the repository is. Multiple code bases can be tracked by defining a separate
project for each.

# COMMANDS

The following *:patch* sub-commands are supported:

*:patch init* [*-f*] [_<project>_]
	Creates a new project _<project>_. If _<project>_ is not defined, *aerc*
	will use the last element of the current directory path. It also
	performs a search for a supported repository in the current directory.

	*-f*: Overwrite an existing project.

*:patch list* [*-a*]++
*:patch ls* [*-a*]
	Lists the current project with the tracked patch sets.

	*-a*: Lists all projects.

*:patch apply* [*-c* _<cmd>_] _<tag>_
	Applies the selected message(s) to the repository of the current
	project. It uses the *:pipe* command for this and keeps track of the
	applied patch.

	A user-defined command for applying patches can be used with the *-c*
	option. Any occurence of '%r' in the command string will be replaced
	with the root directory of the current project. However, this approach
	is not recommended in general and should only be used for very specific
	purposes, i.e. when a maintainer is applying a patch set via a separate
	script to deal with git trailers.

	*aerc* will propose completions for the _<tag>_ based on the subject
	lines of the selected or marked messages.

*:patch remove* _<tag>_
	Removes the patch _<tag>_ from the repository.

*:patch rebase* [_<commit-ish>_]
	Rebases the patch data on commit _<commit-ish>_.

	If the _<commit-ish>_ is omitted, *aerc* will use the base commit of
	the current project for the rebase.

*:patch cd*
	Changes the working directory to the root directory of the current
	project.

*:patch term* [_<cmd>_]
	Opens a shell (or runs _<cmd>_) in the working directory of the
	current project.

*:patch switch* _<project>_
	Switches the context to _<project>_.

*:patch delete* [_<project>_]
	Deletes all patch tracking data for _<project>_. If no project is
	provided, the current project is deleted.

*:patch*
	Root command for path management. Use it to run the sub-commands.


# GETTING STARTED

Make sure you have an initialized project (see *:patch init*).

Now, there are two ways to get patches under the local patch management system:

- Apply patches with the *:patch apply* command. This will automatically create
  a new tag for the applied commits.

- Use *:patch rebase*. If there are some existing local patches in the commit
  history that should be managed by *aerc*, you can run *:patch rebase
  <commit-ish>* and set the _<commit-ish>_ to the commit before the first patch
  that you want to include. For a *git* repository which has an upstream called
  *origin*, you would run *:patch rebase origin/master*.

# EXAMPLE

The following example demonstrates how to manage the local patch sets.

First, a project needs to be initialized. This is done by changing into the
working directory where the project's repository is located. For this example,
let's assume we have a project called _bar_ in the directory
_/home/user/foo/bar_.

```
:cd /home/user/foo/bar
```

and then creating a new project with

```
:patch init
```

If no name is provided to *:patch init*, *aerc* will use the last element of the
working directory path (here: _bar_).

Now the patch tracking is ready for action. Go to the message list, mark a patch
series and apply it:

```
:patch apply fix_v2
```

This will apply the selected patch set and assigns the _fix_v2_ tag to those
commits. The tag helps to keep the commits grouped together, and will be helpful
when we want to remove this exact patch set at a later point.

With *:patch list* you can verify that the patch set was correctly applied.

If there is a change in the underlying repository (e.g. by rebasing to
upstream), the hashes of the applied local commits can change. *:patch list* can
detect such a change and will then propose to rebase the internal data. To
do this, run

```
:patch rebase
```

This will open an editor where you can adjust the correct tags again. You could
also change the rebase point by providing an optional argument (e.g. a commit
hash, or even _HEAD~3_ or _origin/master_, etc.).

To remove a patch set, use the tag that was assigned during applying:

```
:patch remove fix_v2
```

And to delete the project data in *aerc*:

```
:patch delete bar
```

# SUPPORTED REVISION CONTROL SYSTEMS

The supported revision control systems are currently: *git*.

# SEE ALSO

*aerc*(1)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
