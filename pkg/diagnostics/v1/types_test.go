package diagnostics

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	type args struct {
		groupID                  string
		clusterID                string
		ambassadorVersion        string
		errorCheckSystemPass     bool
		mappingCheckSystemPass   bool
		tlsCheckSystemPass       bool
		errorCheckSpecificPass   bool
		mappingCheckSpecificPass bool
		tlsCheckSpecificPass     bool
		noticeLevel              string
		noticeMessage            string
		routeOneURL              string
		routeOneService          string
		routeOneWeight           int
		hColor                   string
	}
	testcases := []struct {
		testName string
		fileName string
		args     args
		want     Diagnostics
	}{
		{
			testName: "should return correct diagnostics struct",
			fileName: "diagnostics.json",
			args: args{
				groupID:                  "8db7c38a2e026c4ebe74ba2e75770b855a1a5437",
				clusterID:                "30defff8-f47d-5c41-a62c-22ecc72f1714",
				ambassadorVersion:        "3.0.0-rc.0",
				errorCheckSystemPass:     false,
				errorCheckSpecificPass:   false,
				mappingCheckSystemPass:   true,
				mappingCheckSpecificPass: true,
				tlsCheckSystemPass:       true,
				tlsCheckSpecificPass:     true,
				noticeLevel:              "NOTICE",
				noticeMessage:            "-global-: A future Ambassador version will change the GRPC protocol version for AuthServices. See the CHANGELOG for details.",
				routeOneURL:              "http://*/ambassador/v0/",
				routeOneService:          "service-1:8500",
				routeOneWeight:           100,
				hColor:                   "orange",
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			contents, err := ioutil.ReadFile(filepath.Join("testdata", tc.fileName))
			if err != nil {
				t.Fatal(err)
			}
			tc.want = Diagnostics{
				Notices: []*Notice{
					{Level: tc.args.noticeLevel, Message: tc.args.noticeMessage},
				},
				RouteInfo: []*RouteInfo{
					{
						Clusters: []*RouteInfoCluster{
							{
								Service: tc.args.routeOneService,
								Weight:  tc.args.routeOneWeight,
								Hcolor:  tc.args.hColor,
							},
						},
						Route: &RouteInfoRoute{
							GroupID: tc.args.groupID,
						},
						Key: tc.args.routeOneURL,
					},
				},
				System: &System{
					ClusterID: tc.args.clusterID,
				},
			}
			var diagnostics Diagnostics
			if err := json.Unmarshal(contents, &diagnostics); err != nil {
				t.Error(err)
			}

			assert.Equal(t, tc.want.Notices[0].Level, diagnostics.Notices[0].Level)
			assert.Equal(t, tc.want.Notices[0].Message, diagnostics.Notices[0].Message)
			assert.Equal(t, tc.want.RouteInfo[0].Clusters[0].Service, diagnostics.RouteInfo[0].Clusters[0].Service)
			assert.Equal(t, tc.want.RouteInfo[0].Clusters[0].Weight, diagnostics.RouteInfo[0].Clusters[0].Weight)
			assert.Equal(t, tc.want.RouteInfo[0].Clusters[0].Hcolor, diagnostics.RouteInfo[0].Clusters[0].Hcolor)
			assert.Equal(t, tc.want.RouteInfo[0].Route.GroupID, diagnostics.RouteInfo[0].Route.GroupID)
			assert.Equal(t, tc.want.RouteInfo[0].Key, diagnostics.RouteInfo[0].Key)
			assert.Equal(t, tc.want.System.ClusterID, diagnostics.System.ClusterID)
		})
	}
}
