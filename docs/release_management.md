# Release Process

Between releases we keep track of notable changes in CHANGELOG.md.

When we want to make a release we should update CHANGELOG.md to contain the release notes for the planned release in a section for
the proposed release number. This update is the commit that will be tagged with as the actual release which ensures that each release
contains a copy of it's own release notes. 

We should also copy the release notes to the Github releases page, but CHANGELOG.md is the primary place to keep the release notes. 

The release commit should be tagged with an annotated and signed tag:

    git tag -asm vx.x.x vx.x.x
    git push --tags

A non-prescriptive example of the release process might look like the following:

```sh
git checkout master
git pull                                # checkout/pull latest master
git checkout -b vx.x.x-release          # create release branch
vi CHANGELOG.md                         # update CHANGELOG.md
make visor                              # validate build
go mod tidy                             # ensure tidy go.mod for release
git add CHANGELOG.md go.mod go.sum
git commit -m "chore(docs): Update CHANGELOG for vx.x.x-rc1"
                                        # commit CHANGELOG/go.mod updates
git tag -asm vx.x.x-rc1 vx.x.x-rc1      # create signed/annotated tag
git push --tags origin vx.x.x-release
                                        # push release branch and tags

# release validation

# optional hotfix flow
git commit -m "fix: Hotfix desc"        # optional hotfixes applied to release branch
vi CHANGELOG.md                         # update CHANGELOG.md
make visor                              # validate build
go mod tidy                             # ensure tidy go.mod for release
git add CHANGELOG.md go.mod go.sum
git commit -m "chore(docs): Update CHANGELOG for vx.x.x-rc2"
git tag -asm vx.x.x-rc2 vx.x.x-rc2
git push --tags origin vx.x.x-release   # push hotfix and new release candidate tag

# release acceptance

vi CHANGELOG.md
git add CHANGELOG.md
git commit -m "chore(docs): Update CHANGELOG for vx.x.x"
                                        # update/add/commit CHANGELOG.md
git tag -asm vx.x.x vx.x.x
git push --tags origin vx.x.x-release   # tag and push final release

git merge master                        # resolve upstream changes within release branch
git push origin vx.x.x-release          # push merge resolution

git checkout master
git merge vx.x.x-release
git push origin master                  # clean merge commit of release branch into master and push
```

NOTE: `sentinel-visor` pull requests prefer to be squash-merged into `master`, however considering this workflow tags release candidate within the release branch which we want to easily resolve in the repository's history, it is preferred to not squash and instead merge the release branch into `master`.


## Updating CHANGELOG.md

The format is a variant of [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) combined with categories from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/). This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). The [github.com/git-chglog](https://github.com/git-chglog/git-chglog) utility assists us with maintaing CHANGELOG.md.

The sections within each release have a preferred order which prioritizes by largest-user-impact-first: `Feat > Refactor > Fix > {Area-specific or Custom Sections} > Chore`

Here is an example workflow of how CHANGELOG.md might be updated.

```sh
# checkout master and pull latest changes
git checkout master
git pull

# output the CHANGELOG content for the next release (assuming next release is v0.5.0-rc1)
go run github.com/git-chglog/git-chglog/cmd/git-chglog -o CHANGELOG_updates.md --next-tag v0.5.0-rc1

# reconcile CHANGELOG_updates.md into CHANGELOG.md applying the preferred section order
vi CHANGELOG*.md

# commit changes
rm CHANGELOG_updates.md
git add CHANGELOG.md
git commit -m 'chore(docs): Update CHANGELOG for v0.5.0-rc1'
```

Here is an [example of how the diff might look](https://github.com/filecoin-project/sentinel-visor/pull/326/commits/9536df9e39991a3b78013d1d1b36fef94562556d).
