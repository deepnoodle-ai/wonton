package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestAnimation(t *testing.T) {
	t.Run("Basic Lifecycle", func(t *testing.T) {
		anim := NewAnimation(10) // 10 frames
		assert.False(t, anim.IsRunning(), "New animation should not be running")

		anim.Start(100)
		assert.True(t, anim.IsRunning(), "Started animation should be running")

		// Frame 100 (start) -> progress 0.0
		anim.Update(100)
		assert.Equal(t, 0.0, anim.Value())

		// Frame 105 (mid) -> progress 0.5
		anim.Update(105)
		assert.Equal(t, 0.5, anim.Value())

		// Frame 110 (end) -> progress 1.0
		anim.Update(110)
		assert.Equal(t, 1.0, anim.Value())

		// Animation stays running at 1.0 until it ticks over
		assert.True(t, anim.IsRunning(), "Animation should still be running at exactly duration")

		// Frame 111 (past end) -> progress 1.0, stops running
		anim.Update(111)
		assert.Equal(t, 1.0, anim.Value())
		assert.False(t, anim.IsRunning(), "Finished animation should not be running")
	})

	t.Run("Looping", func(t *testing.T) {
		anim := NewAnimation(10).WithLoop(true)
		anim.Start(0)

		// Frame 15 -> progress 0.5 (second loop)
		anim.Update(15)
		assert.Equal(t, 0.5, anim.Value())
		assert.True(t, anim.IsRunning(), "Looping animation should still be running")
	})

	t.Run("PingPong", func(t *testing.T) {
		anim := NewAnimation(10).WithPingPong(true)
		anim.Start(0)

		// Frame 5 -> 0.5 (forward)
		anim.Update(5)
		assert.Equal(t, 0.5, anim.Value())

		// Frame 15 -> 0.5 (backward: 1.0 - 0.5)
		anim.Update(15)
		assert.Equal(t, 0.5, anim.Value())

		// Frame 25 -> 0.5 (forward again)
		anim.Update(25)
		assert.Equal(t, 0.5, anim.Value())
	})

	t.Run("Lerp", func(t *testing.T) {
		anim := NewAnimation(10)
		anim.Start(0)
		anim.Update(5) // 0.5

		assert.Equal(t, 15.0, anim.Lerp(10, 20))
	})

	t.Run("LerpRGB", func(t *testing.T) {
		anim := NewAnimation(10)
		anim.Start(0)
		anim.Update(5) // 0.5

		c1 := NewRGB(0, 0, 0)
		c2 := NewRGB(100, 200, 50)
		got := anim.LerpRGB(c1, c2)
		want := NewRGB(50, 100, 25)

		assert.Equal(t, want, got)
	})
}

func TestAnimationSequence(t *testing.T) {
	a1 := NewAnimation(10)
	a2 := NewAnimation(10)
	seq := NewAnimationSequence(a1, a2)

	// Start sequence implicitly via Update
	seq.Update(0)

	// Frame 5: a1 is at 0.5
	seq.Update(5)
	assert.Equal(t, 0.5, seq.Value())

	// Frame 10: a1 is at 1.0, still running
	seq.Update(10)
	assert.Equal(t, 1.0, seq.Value())

	// Frame 11: a1 stops. Index increments to a2. a2 NOT started yet.
	seq.Update(11)
	assert.Equal(t, 0.0, seq.Value())

	// Frame 12: a2 starts (startFrame=12). Elapsed=0.
	seq.Update(12)
	assert.Equal(t, 0.0, seq.Value())

	// Frame 17: a2 (start=12). Elapsed=5. Progress=0.5.
	seq.Update(17)
	assert.Equal(t, 0.5, seq.Value())

	// Frame 22: a2 (start=12). Elapsed=10. Progress=1.0.
	seq.Update(22)
	assert.Equal(t, 1.0, seq.Value())

	// Frame 23: a2 stops. Complete.
	seq.Update(23)
	assert.True(t, seq.IsComplete(), "Sequence should be complete")

	seq.Reset()
	assert.False(t, seq.IsComplete(), "Sequence should not be complete after reset")
}

func TestAnimationGroup(t *testing.T) {
	a1 := NewAnimation(10)
	a2 := NewAnimation(20)
	group := NewAnimationGroup(a1, a2)

	group.Start(0)
	group.Update(5)

	// a1: 0.5, a2: 0.25
	// Average: 0.375
	assert.Equal(t, 0.375, group.Value())

	group.CombineMax()
	assert.Equal(t, 0.5, group.Value())

	group.CombineMin()
	assert.Equal(t, 0.25, group.Value())
}

func TestAnimationController(t *testing.T) {
	ac := NewAnimationController()
	anim := NewAnimation(10)

	ac.Register("test", anim)
	assert.True(t, ac.Get("test") == anim, "Failed to retrieve registered animation")

	anim.Start(0)
	ac.Update(5)

	assert.Equal(t, 0.5, anim.Value())
	assert.Equal(t, uint64(5), ac.GlobalTime())
}
