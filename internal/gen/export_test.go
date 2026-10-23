package gen

// Test helpers of package gen exposed to the external gen_test package, whose
// tests import packages that themselves import gen.
var (
	// SnapshotTree is snapshotTree.
	SnapshotTree = snapshotTree
	// WritePlanFile is writePlanFile.
	WritePlanFile = writePlanFile
	// SymlinkPlan is symlinkPlan.
	SymlinkPlan = symlinkPlan
	// ProviderPlan is providerPlan.
	ProviderPlan = providerPlan
	// Update is the -update flag.
	Update = update
)
