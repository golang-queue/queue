package queue

func ExampleNewEmptyLogger() {
	l := NewEmptyLogger()
	l.Info("test")
	l.Infof("test")
	l.Error("test")
	l.Errorf("test")
	l.Fatal("test")
	l.Fatalf("test")
	// Output:
}
