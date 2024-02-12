package queue

func ExampleNewEmptyLogger() {
	l := NewEmptyLogger()
	l.Info("test")
	l.Error("test")
	l.Fatal("test")
	l.Warn("test")
	l.Debug("test")
	// Output:
}
