package replica_set_tls

import (
	"testing"
	"time"

	. "github.com/mongodb/mongodb-kubernetes-operator/test/e2e/util/mongotester"

	"github.com/mongodb/mongodb-kubernetes-operator/test/e2e/tlstests"

	e2eutil "github.com/mongodb/mongodb-kubernetes-operator/test/e2e"
	"github.com/mongodb/mongodb-kubernetes-operator/test/e2e/mongodbtests"
	setup "github.com/mongodb/mongodb-kubernetes-operator/test/e2e/setup"
	f "github.com/operator-framework/operator-sdk/pkg/test"
)

func TestMain(m *testing.M) {
	f.MainEntry(m)
}

func TestReplicaSetTLSUpgrade(t *testing.T) {
	ctx, shouldCleanup := setup.InitTest(t)
	if shouldCleanup {
		defer ctx.Cleanup()
	}

	mdb, user := e2eutil.NewTestMongoDB("mdb-tls")
	_, err := setup.GeneratePasswordForUser(user, ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := setup.CreateTLSResources(mdb.Namespace, ctx); err != nil {
		t.Fatalf("Failed to set up TLS resources: %s", err)
	}

	tester, err := FromResource(t, mdb)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Create MongoDB Resource", mongodbtests.CreateMongoDBResource(&mdb, ctx))
	t.Run("Basic tests", mongodbtests.BasicFunctionality(&mdb))
	t.Run("Test Basic Connectivity", tester.ConnectivitySucceeds(WithoutTls()))
	t.Run("Ensure Authentication", tester.EnsureAuthenticationIsConfigured(3))

	// Enable TLS as optional
	t.Run("MongoDB is reachable while TLS is being enabled", func(t *testing.T) {
		defer tester.StartBackgroundConnectivityTest(t, time.Second*10, WithoutTls())()
		t.Run("Upgrade to TLS", tlstests.EnableTLS(&mdb, true))
		t.Run("Stateful Set Reaches Ready State, after enabling TLS", mongodbtests.StatefulSetIsReady(&mdb))
		t.Run("Wait for TLS to be enabled", tester.HasTlsMode("preferSSL", 60, WithoutTls()))
	})

	// Ensure MongoDB is reachable both with and without TLS
	t.Run("Test Basic Connectivity", tester.ConnectivitySucceeds(WithoutTls()))
	t.Run("Test Basic TLS Connectivity", tester.ConnectivitySucceeds(WithTls()))
	t.Run("Internal cluster keyfile authentication is enabled", tester.HasKeyfileAuth(3, WithTls()))

	// Make TLS required
	t.Run("MongoDB is reachable over TLS while making TLS required", func(t *testing.T) {
		defer tester.StartBackgroundConnectivityTest(t, time.Second*10, WithTls())()
		t.Run("Make TLS required", tlstests.EnableTLS(&mdb, false))
		t.Run("Wait for TLS to be required", tester.HasTlsMode("requireSSL", 120, WithTls()))
	})

	// Ensure MongoDB is reachable only over TLS
	t.Run("Test Basic TLS Connectivity", tester.ConnectivitySucceeds(WithTls()))
	t.Run("Test TLS Required For Connectivity", tester.ConnectivityFails(WithoutTls()))
}
