package updater

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/blang/semver"
)

//Artifact map type to custom unmarshal it
type ArtifactMap map[string]Artifact

//evertime a artifact is found in the xml it gets decoded using this function, it
//unmarshals the artifact struct and stores it in the map
func (am ArtifactMap) UnmarshalXML(e *xml.Decoder, start xml.StartElement) error {
	a := Artifact{}
	err := e.DecodeElement(&a, &start)
	if err != nil {
		return err
	}
	am[a.Id] = a
	return nil
}

//Semver type to be able to custom unmarshal it
type Version struct {
	semver.Version
}

//Get the version string an create a semver from it
func (v *Version) UnmarshalXMLAttr(attr xml.Attr) error {
	str := attr.Value
	parsed, err := semver.New(str)
	if err != nil {
		return err
	}
	v.Version = parsed

	return nil
}

//Collection of artifacts
type ReleaseDescriptor struct {
	XMLName   xml.Name    `xml:"releaseDescriptor"`
	Href      string      `xml:"href,attr"`    //href where to get this descriptor
	Version   Version     `xml:"version,attr"` //version of the this release
	Artifacts ArtifactMap `xml:"artifact"`     //artifacts associated to this descriptor, the key is the artifact id
}

//Create a new descriptor from all the info
func NewReleaseDescriptor(href string, version string, artifacts ...Artifact) (rd ReleaseDescriptor, err error) {
	sver, err := semver.Parse(version)
	if err != nil {
		return
	}
	rd = ReleaseDescriptor{
		Href:      href,
		Version:   Version{sver},
		Artifacts: map[string]Artifact{},
	}
	for _, a := range artifacts {
		rd.Artifacts[a.Id] = a
	}
	return

}

//Create a new empty relase descriptor
func NewEmptyReleaseDescriptor() ReleaseDescriptor {
	return ReleaseDescriptor{
		Artifacts: map[string]Artifact{},
	}

}

//Compares two indeces Returning a list of differences
func (i ReleaseDescriptor) IsDiff(old ReleaseDescriptor) (is bool, diffs DiffSet) {
	//no changes
	if i.Version.Equals(old.Version.Version) {

		return
	}

	news := i.Artifacts
	olds := old.Artifacts
	//range the new artifacts to find differences
	for id, n := range news {
		newArt := n
		oldArt, ok := olds[id]
		//there's no old version
		if !ok {
			diffs = append(diffs, Diff{New: &newArt, Old: nil})
		} else if newArt.Version != oldArt.Version {
			diffs = append(diffs, Diff{New: &newArt, Old: &oldArt})
		}

	}
	//range the old artifacts to find deleted artifacts
	for id, o := range olds {
		if _, ok := news[id]; !ok {
			oldArt := o
			diffs = append(diffs, Diff{New: nil, Old: &oldArt})
		}

	}
	return true, diffs
}

func (r ReleaseDescriptor) UpdateFrom(local ReleaseDescriptor, installationPath string) error {
	changes, diffSet := r.IsDiff(local)
	if !changes {
		//nothing to do!
		return nil
	}
	toDeploy, err := Download(os.TempDir(), diffSet.ToDownload()...)
	if err != nil {
		return err
	}
	ok, errs := Remove(diffSet.ToRemove(installationPath))
	if !ok {
		//warn
		fmt.Printf("errs %+v\n", errs)
	}
	ok, errs = Deploy(toDeploy, installationPath)
	if !ok {
		//warn
		fmt.Printf("errs %+v\n", errs)
	}
	//clean up the tmp dir
	ok, errs = Remove(toDeploy)
	if !ok {
		//warn
		fmt.Printf("errs %+v\n", errs)
	}
	return nil
}

//String for logging
func (rd ReleaseDescriptor) String() string {
	return fmt.Sprintf("ReleaseDescriptor %v", rd.Version.String())
}
