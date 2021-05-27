package gitaly

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly/pktline"
)

const (
	oid1 = "78fb81a02b03f0013360292ec5106763af32c287"
	oid2 = "0f6394307cd7d4909be96a0c818d8094a4cb0e5b"
)

func TestSingleRefParses(t *testing.T) {
	buf := &bytes.Buffer{}
	pktline.WriteString(buf, "# service=git-upload-pack\n")
	pktline.WriteFlush(buf)
	pktline.WriteString(buf, oid1+" HEAD\x00capability")
	pktline.WriteFlush(buf)

	d, err := ParseReferenceDiscovery(buf)
	require.NoError(t, err)
	require.Equal(t, []string{"capability"}, d.Caps)
	require.Equal(t, []Reference{{Oid: oid1, Name: "HEAD"}}, d.Refs)
}

func TestMultipleRefsAndCapsParse(t *testing.T) {
	buf := &bytes.Buffer{}
	pktline.WriteString(buf, "# service=git-upload-pack\n")
	pktline.WriteFlush(buf)
	pktline.WriteString(buf, oid1+" HEAD\x00first second")
	pktline.WriteString(buf, oid2+" refs/heads/master")
	pktline.WriteFlush(buf)

	d, err := ParseReferenceDiscovery(buf)
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second"}, d.Caps)
	require.Equal(t, []Reference{{Oid: oid1, Name: "HEAD"}, {Oid: oid2, Name: "refs/heads/master"}}, d.Refs)
}

func TestInvalidHeaderFails(t *testing.T) {
	buf := &bytes.Buffer{}
	pktline.WriteString(buf, "# service=invalid\n")
	pktline.WriteFlush(buf)
	pktline.WriteString(buf, oid1+" HEAD\x00caps")
	pktline.WriteFlush(buf)

	_, err := ParseReferenceDiscovery(buf)
	require.Error(t, err)
}

func TestMissingRefsFail(t *testing.T) {
	buf := &bytes.Buffer{}
	pktline.WriteString(buf, "# service=git-upload-pack\n")
	pktline.WriteFlush(buf)
	pktline.WriteFlush(buf)

	_, err := ParseReferenceDiscovery(buf)
	require.Error(t, err)
}

func TestInvalidRefFail(t *testing.T) {
	buf := &bytes.Buffer{}
	pktline.WriteString(buf, "# service=git-upload-pack\n")
	pktline.WriteFlush(buf)
	pktline.WriteString(buf, oid1+" HEAD\x00caps")
	pktline.WriteString(buf, oid2)
	pktline.WriteFlush(buf)

	_, err := ParseReferenceDiscovery(buf)
	require.Error(t, err)
}

func TestMissingTrailingFlushFails(t *testing.T) {
	buf := &bytes.Buffer{}
	pktline.WriteString(buf, "# service=git-upload-pack\n")
	pktline.WriteFlush(buf)
	pktline.WriteString(buf, oid1+" HEAD\x00caps")

	d := ReferenceDiscovery{}
	require.Error(t, d.Parse(buf))
}
