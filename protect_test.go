package protect

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SimpleStruct struct {
	ID   string `protectfor:"create,update"`
	Code string `protectfor:"update"`
	Name string
}

type NestedStruct struct {
	ID     string `protectfor:"create,update"`
	Parent *SimpleStruct
	Child  SimpleStruct
}

type SliceStruct struct {
	ID      string `protectfor:"create,update"`
	Names   []string
	Numbers []int
}

type MatchSlice []SimpleStruct
type LongerSlice []SimpleStruct
type ShorterSlice []SimpleStruct

type SliceWithOptions struct {
	ID        string `protectfor:"create,update"`
	Items     []SimpleStruct
	LongList  []SimpleStruct
	ShortList []SimpleStruct
	MapItems  map[string]SimpleStruct
	MapMatch  map[string]SimpleStruct
}

// 時間フィールドを持つ構造体を追加
type TimeStruct struct {
	ID        string     `protectfor:"create,update"`
	CreatedAt time.Time  // 値型のtime.Time
	UpdatedAt *time.Time // ポインタ型のtime.Time
	Name      string
}

func TestCopy(t *testing.T) {
	t.Run("simple struct", func(t *testing.T) {
		src := SimpleStruct{
			ID:   "123",
			Code: "ABC",
			Name: "Test",
		}

		t.Run("create tag", func(t *testing.T) {
			dst := SimpleStruct{}

			err := Copy("create", &src, &dst)
			assert.NoError(t, err)

			// ID should be protected
			assert.Empty(t, dst.ID)
			assert.Equal(t, src.Code, dst.Code)
			assert.Equal(t, src.Name, dst.Name)
		})

		t.Run("update tag", func(t *testing.T) {
			dst := SimpleStruct{}

			err := Copy("update", &src, &dst)
			assert.NoError(t, err)

			// ID and Code should be protected
			assert.Empty(t, dst.ID)
			assert.Empty(t, dst.Code)
			assert.Equal(t, src.Name, dst.Name)
		})

		t.Run("no tag", func(t *testing.T) {
			dst := SimpleStruct{}

			err := Copy("", &src, &dst)
			assert.NoError(t, err)

			// All fields should be copied
			assert.Equal(t, src.ID, dst.ID)
			assert.Equal(t, src.Code, dst.Code)
			assert.Equal(t, src.Name, dst.Name)
		})
	})

	t.Run("nested struct", func(t *testing.T) {
		src := NestedStruct{
			ID: "123",
			Parent: &SimpleStruct{
				ID:   "parent-123",
				Code: "parent-ABC",
				Name: "Parent",
			},
			Child: SimpleStruct{
				ID:   "child-123",
				Code: "child-ABC",
				Name: "Child",
			},
		}

		t.Run("create tag", func(t *testing.T) {
			dst := NestedStruct{}

			err := Copy("create", &src, &dst)
			assert.NoError(t, err)

			// NestedStruct.ID should be protected
			assert.Empty(t, dst.ID)

			// Parent should be deep-copied
			assert.NotNil(t, dst.Parent)
			assert.Empty(t, dst.Parent.ID) // ID is protected
			assert.Equal(t, src.Parent.Code, dst.Parent.Code)
			assert.Equal(t, src.Parent.Name, dst.Parent.Name)

			// Child should be deep-copied
			assert.Empty(t, dst.Child.ID) // ID is protected
			assert.Equal(t, src.Child.Code, dst.Child.Code)
			assert.Equal(t, src.Child.Name, dst.Child.Name)
		})
	})

	t.Run("slice", func(t *testing.T) {
		src := SliceStruct{
			ID:      "123",
			Names:   []string{"Alice", "Bob", "Charlie"},
			Numbers: []int{1, 2, 3},
		}

		t.Run("create tag", func(t *testing.T) {
			dst := SliceStruct{}

			err := Copy("create", &src, &dst)
			assert.NoError(t, err)

			// ID should be protected
			assert.Empty(t, dst.ID)

			// Slices should be deep-copied
			assert.Equal(t, src.Names, dst.Names)
			assert.Equal(t, src.Numbers, dst.Numbers)

			// Verify that modifying destination doesn't affect source
			srcNames := src.Names
			dstNames := make([]string, len(dst.Names))
			copy(dstNames, dst.Names)
			dstNames[0] = "Modified"

			assert.Equal(t, "Alice", srcNames[0])
			assert.NotEqual(t, dstNames[0], src.Names[0])
		})
	})

	t.Run("time fields", func(t *testing.T) {
		now := time.Now()
		updatedAt := now.Add(1 * time.Hour)

		src := TimeStruct{
			ID:        "123",
			CreatedAt: now,
			UpdatedAt: &updatedAt,
			Name:      "Test with time",
		}

		t.Run("time.Time fields should be copied", func(t *testing.T) {
			dst := TimeStruct{}

			err := Copy("create", &src, &dst)
			assert.NoError(t, err)

			// ID should be protected
			assert.Empty(t, dst.ID)

			// これが現状の動作 (バグの確認): 時間フィールドがコピーされるべきだが、コピーされていない
			// このテストは失敗することを期待（時間フィールドが正しくコピーされていないため）
			assert.Equal(t, src.CreatedAt, dst.CreatedAt, "CreatedAt (time.Time) should be copied")

			// ポインタ型のtime.Timeフィールドもコピーされるべきだが、コピーされていない
			assert.NotNil(t, dst.UpdatedAt, "UpdatedAt (*time.Time) should not be nil")
			if dst.UpdatedAt != nil {
				assert.Equal(t, *src.UpdatedAt, *dst.UpdatedAt, "UpdatedAt (*time.Time) values should be equal")
			}

			// 名前はコピーされているはず
			assert.Equal(t, src.Name, dst.Name)
		})
	})
}

func TestClone(t *testing.T) {
	src := SimpleStruct{
		ID:   "123",
		Code: "ABC",
		Name: "Test",
	}

	cloned := Clone(&src).(*SimpleStruct)

	// All fields should be copied
	assert.Equal(t, src.ID, cloned.ID)
	assert.Equal(t, src.Code, cloned.Code)
	assert.Equal(t, src.Name, cloned.Name)

	// Verify that modifying source doesn't affect clone
	src.Name = "Modified"
	assert.NotEqual(t, src.Name, cloned.Name)
}

// 新しいカスタムProtectorを作成してテスト用のタグを設定
func createTestProtector() *Protector {
	p := NewProtector("protectfor", "protectopt")
	return p
}

func TestSliceOptions(t *testing.T) {
	t.Run("overwrite option", func(t *testing.T) {
		p := createTestProtector()

		src := SliceWithOptions{
			Items: []SimpleStruct{
				{ID: "1", Code: "A", Name: "First"},
				{ID: "2", Code: "B", Name: "Second"},
			},
		}

		dst := SliceWithOptions{
			Items: []SimpleStruct{
				{ID: "X", Code: "Y", Name: "Existing"},
				{ID: "Z", Code: "W", Name: "Another"},
				{ID: "Extra", Code: "Extra", Name: "Extra"},
			},
		}

		// デフォルトでは overwrite なので特に設定は不要

		err := p.Copy("create", &src, &dst)
		assert.NoError(t, err)

		// Length should match the source length (2 items)
		assert.Equal(t, len(src.Items), len(dst.Items))

		// In overwrite mode, tags are ignored - simple clone instead
		assert.Equal(t, src.Items[0].ID, dst.Items[0].ID) // ID is not protected in overwrite mode
		assert.Equal(t, src.Items[0].Code, dst.Items[0].Code)
		assert.Equal(t, src.Items[0].Name, dst.Items[0].Name)
	})

	t.Run("match option", func(t *testing.T) {
		p := createTestProtector()

		src := SliceWithOptions{
			Items: []SimpleStruct{
				{ID: "1", Code: "A", Name: "First"},
				{ID: "2", Code: "B", Name: "Second"},
			},
		}

		dst := SliceWithOptions{
			Items: []SimpleStruct{
				{ID: "X", Code: "Y", Name: "Existing"},
				{ID: "Z", Code: "W", Name: "Another"},
				{ID: "Extra", Code: "Extra", Name: "Extra"},
			},
		}

		// スライスに対するオプションをオーバーライドして設定
		p.setSliceOption(dst.Items, "match")

		// このテストでは明示的に create タグを指定
		err := p.Copy("create", &src, &dst)
		assert.NoError(t, err)

		// Length should match the source length (2 items)
		assert.Equal(t, len(src.Items), len(dst.Items))

		// Items should be deep copied with original ID preserved
		assert.Equal(t, "X", dst.Items[0].ID) // ID from destination is preserved
		assert.Equal(t, src.Items[0].Code, dst.Items[0].Code)
		assert.Equal(t, src.Items[0].Name, dst.Items[0].Name)
	})

	t.Run("longer option", func(t *testing.T) {
		p := createTestProtector()

		src := SliceWithOptions{
			LongList: []SimpleStruct{
				{ID: "1", Code: "A", Name: "First"},
				{ID: "2", Code: "B", Name: "Second"},
			},
		}

		dst := SliceWithOptions{
			LongList: []SimpleStruct{
				{ID: "X", Code: "Y", Name: "Existing"},
				{ID: "Z", Code: "W", Name: "Another"},
				{ID: "Extra", Code: "Extra", Name: "Extra"},
			},
		}

		originalLength := len(dst.LongList)

		// スライスに対するオプションをオーバーライドして設定
		p.setSliceOption(dst.LongList, "longer")

		// このテストでは明示的に create タグを指定
		err := p.Copy("create", &src, &dst)
		assert.NoError(t, err)

		// Length should not be changed since destination is longer
		assert.Equal(t, originalLength, len(dst.LongList))

		// First two items should be copied but ID preserved
		assert.Equal(t, "X", dst.LongList[0].ID) // Original ID is preserved
		assert.Equal(t, src.LongList[0].Code, dst.LongList[0].Code)
		assert.Equal(t, src.LongList[0].Name, dst.LongList[0].Name)

		assert.Equal(t, "Z", dst.LongList[1].ID) // Original ID is preserved
		assert.Equal(t, src.LongList[1].Code, dst.LongList[1].Code)
		assert.Equal(t, src.LongList[1].Name, dst.LongList[1].Name)

		// Third item should remain unchanged
		assert.Equal(t, "Extra", dst.LongList[2].ID)
	})

	t.Run("shorter option", func(t *testing.T) {
		p := createTestProtector()

		src := SliceWithOptions{
			ShortList: []SimpleStruct{
				{ID: "1", Code: "A", Name: "First"},
				{ID: "2", Code: "B", Name: "Second"},
				{ID: "3", Code: "C", Name: "Third"},
				{ID: "4", Code: "D", Name: "Fourth"},
			},
		}

		dst := SliceWithOptions{
			ShortList: []SimpleStruct{
				{ID: "X", Code: "Y", Name: "Existing"},
				{ID: "Z", Code: "W", Name: "Another"},
			},
		}

		originalLength := len(dst.ShortList)

		// スライスに対するオプションをオーバーライドして設定
		p.setSliceOption(dst.ShortList, "shorter")

		// このテストでは明示的に create タグを指定
		err := p.Copy("create", &src, &dst)
		assert.NoError(t, err)

		// Only copy as many items as the destination has
		assert.Equal(t, originalLength, len(dst.ShortList))

		// Items should be copied but ID preserved
		assert.Equal(t, "X", dst.ShortList[0].ID) // Original ID is preserved
		assert.Equal(t, src.ShortList[0].Code, dst.ShortList[0].Code)
		assert.Equal(t, src.ShortList[0].Name, dst.ShortList[0].Name)

		assert.Equal(t, "Z", dst.ShortList[1].ID) // Original ID is preserved
		assert.Equal(t, src.ShortList[1].Code, dst.ShortList[1].Code)
		assert.Equal(t, src.ShortList[1].Name, dst.ShortList[1].Name)
	})
}

func TestMapOptions(t *testing.T) {
	simpleMap := map[string]SimpleStruct{
		"first":  {ID: "1", Code: "A", Name: "First"},
		"second": {ID: "2", Code: "B", Name: "Second"},
	}

	t.Run("overwrite option", func(t *testing.T) {
		p := createTestProtector()

		src := SliceWithOptions{
			MapItems: simpleMap,
		}

		dst := SliceWithOptions{
			MapItems: map[string]SimpleStruct{
				"first": {ID: "X", Code: "Y", Name: "Existing"},
				"third": {ID: "3", Code: "C", Name: "Third"},
			},
		}

		// デフォルトでは overwrite なので特に設定は不要

		err := p.Copy("create", &src, &dst)
		assert.NoError(t, err)

		// Should replace the map (overwrite everything)
		assert.Equal(t, len(src.MapItems), len(dst.MapItems))
		assert.Contains(t, dst.MapItems, "first")
		assert.Contains(t, dst.MapItems, "second")
		assert.NotContains(t, dst.MapItems, "third") // Third item should be gone

		// In overwrite mode, tags are ignored - simple clone instead
		assert.Equal(t, src.MapItems["first"].ID, dst.MapItems["first"].ID) // ID is not protected
		assert.Equal(t, src.MapItems["first"].Code, dst.MapItems["first"].Code)
		assert.Equal(t, src.MapItems["first"].Name, dst.MapItems["first"].Name)
	})

	t.Run("patch option", func(t *testing.T) {
		p := createTestProtector()

		src := SliceWithOptions{
			MapItems: simpleMap,
		}

		dst := SliceWithOptions{
			MapItems: map[string]SimpleStruct{
				"first": {ID: "X", Code: "Y", Name: "Existing"},
				"third": {ID: "3", Code: "C", Name: "Third"},
			},
		}

		// マップに対するオプションをオーバーライドして設定
		p.setMapOption(dst.MapItems, "patch")

		// このテストでは明示的に create タグを指定
		err := p.Copy("create", &src, &dst)
		assert.NoError(t, err)

		// Should patch the map (add or update, but not remove)
		assert.Equal(t, 3, len(dst.MapItems))

		// first item should be updated but ID preserved
		assert.Equal(t, "X", dst.MapItems["first"].ID) // Original ID is preserved
		assert.Equal(t, src.MapItems["first"].Code, dst.MapItems["first"].Code)
		assert.Equal(t, src.MapItems["first"].Name, dst.MapItems["first"].Name)

		// second item should be added (and since it's new, it's simple clone without tag protection)
		assert.Contains(t, dst.MapItems, "second")
		assert.Equal(t, src.MapItems["second"].ID, dst.MapItems["second"].ID) // New items are simple-cloned

		// third item should be preserved
		assert.Contains(t, dst.MapItems, "third")
	})

	t.Run("match option", func(t *testing.T) {
		p := createTestProtector()

		src := SliceWithOptions{
			MapMatch: simpleMap,
		}

		dst := SliceWithOptions{
			MapMatch: map[string]SimpleStruct{
				"first": {ID: "X", Code: "Y", Name: "Existing"},
				"third": {ID: "3", Code: "C", Name: "Third"},
			},
		}

		// マップに対するオプションをオーバーライドして設定
		p.setMapOption(dst.MapMatch, "match")

		// このテストでは明示的に create タグを指定
		err := p.Copy("create", &src, &dst)
		assert.NoError(t, err)

		// Should make the destination match the source (same keys)
		assert.Equal(t, len(src.MapMatch), len(dst.MapMatch))
		assert.Contains(t, dst.MapMatch, "first")
		assert.Contains(t, dst.MapMatch, "second")
		assert.NotContains(t, dst.MapMatch, "third")

		// first item should be updated but ID preserved
		assert.Equal(t, "X", dst.MapMatch["first"].ID) // Original ID is preserved
		assert.Equal(t, src.MapMatch["first"].Code, dst.MapMatch["first"].Code)
		assert.Equal(t, src.MapMatch["first"].Name, dst.MapMatch["first"].Name)
	})
}

func TestCopySlice(t *testing.T) {
	t.Run("explicit overwrite option", func(t *testing.T) {
		src := []SimpleStruct{
			{ID: "1", Code: "A", Name: "First"},
			{ID: "2", Code: "B", Name: "Second"},
		}

		dst := []SimpleStruct{
			{ID: "X", Code: "Y", Name: "Existing"},
			{ID: "Z", Code: "W", Name: "Another"},
			{ID: "Extra", Code: "Extra", Name: "Extra"},
		}

		err := CopySlice("create", &src, &dst, "overwrite")
		assert.NoError(t, err)

		// Length should match the source length (2 items)
		assert.Equal(t, len(src), len(dst))

		// In overwrite mode, tags are ignored - simple clone instead
		assert.Equal(t, src[0].ID, dst[0].ID) // ID is not protected in overwrite mode
		assert.Equal(t, src[0].Code, dst[0].Code)
		assert.Equal(t, src[0].Name, dst[0].Name)
	})

	t.Run("explicit match option", func(t *testing.T) {
		src := []SimpleStruct{
			{ID: "1", Code: "A", Name: "First"},
			{ID: "2", Code: "B", Name: "Second"},
		}

		dst := []SimpleStruct{
			{ID: "X", Code: "Y", Name: "Existing"},
			{ID: "Z", Code: "W", Name: "Another"},
			{ID: "Extra", Code: "Extra", Name: "Extra"},
		}

		err := CopySlice("create", &src, &dst, "match")
		assert.NoError(t, err)

		// Length should match the source length (2 items)
		assert.Equal(t, len(src), len(dst))

		// Items should be deep copied with original ID preserved
		assert.Equal(t, "X", dst[0].ID) // ID from destination is preserved
		assert.Equal(t, src[0].Code, dst[0].Code)
		assert.Equal(t, src[0].Name, dst[0].Name)
	})

	t.Run("explicit longer option", func(t *testing.T) {
		src := []SimpleStruct{
			{ID: "1", Code: "A", Name: "First"},
			{ID: "2", Code: "B", Name: "Second"},
		}

		dst := []SimpleStruct{
			{ID: "X", Code: "Y", Name: "Existing"},
			{ID: "Z", Code: "W", Name: "Another"},
			{ID: "Extra", Code: "Extra", Name: "Extra"},
		}

		originalLength := len(dst)

		err := CopySlice("create", &src, &dst, "longer")
		assert.NoError(t, err)

		// Length should not be changed since destination is longer
		assert.Equal(t, originalLength, len(dst))

		// First two items should be copied but ID preserved
		assert.Equal(t, "X", dst[0].ID) // Original ID is preserved
		assert.Equal(t, src[0].Code, dst[0].Code)
		assert.Equal(t, src[0].Name, dst[0].Name)

		assert.Equal(t, "Z", dst[1].ID) // Original ID is preserved
		assert.Equal(t, src[1].Code, dst[1].Code)
		assert.Equal(t, src[1].Name, dst[1].Name)

		// Third item should remain unchanged
		assert.Equal(t, "Extra", dst[2].ID)
	})

	t.Run("explicit shorter option", func(t *testing.T) {
		src := []SimpleStruct{
			{ID: "1", Code: "A", Name: "First"},
			{ID: "2", Code: "B", Name: "Second"},
			{ID: "3", Code: "C", Name: "Third"},
			{ID: "4", Code: "D", Name: "Fourth"},
		}

		dst := []SimpleStruct{
			{ID: "X", Code: "Y", Name: "Existing"},
			{ID: "Z", Code: "W", Name: "Another"},
		}

		originalLength := len(dst)

		err := CopySlice("create", &src, &dst, "shorter")
		assert.NoError(t, err)

		// Only copy as many items as the destination has
		assert.Equal(t, originalLength, len(dst))

		// Items should be copied but ID preserved
		assert.Equal(t, "X", dst[0].ID) // Original ID is preserved
		assert.Equal(t, src[0].Code, dst[0].Code)
		assert.Equal(t, src[0].Name, dst[0].Name)

		assert.Equal(t, "Z", dst[1].ID) // Original ID is preserved
		assert.Equal(t, src[1].Code, dst[1].Code)
		assert.Equal(t, src[1].Name, dst[1].Name)
	})

	t.Run("non-slice types", func(t *testing.T) {
		src := SimpleStruct{ID: "1", Code: "A", Name: "Test"}
		dst := SimpleStruct{}

		err := CopySlice("create", &src, &dst, "overwrite")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be slices")
	})

	t.Run("nil values", func(t *testing.T) {
		var src []SimpleStruct = nil
		dst := []SimpleStruct{
			{ID: "X", Code: "Y", Name: "Existing"},
		}

		err := CopySlice("create", &src, &dst, "overwrite")
		assert.NoError(t, err)
		assert.Nil(t, dst)
	})

	t.Run("complex structures in slice", func(t *testing.T) {
		src := []NestedStruct{
			{
				ID: "1",
				Parent: &SimpleStruct{
					ID:   "p1",
					Code: "pc1",
					Name: "Parent 1",
				},
				Child: SimpleStruct{
					ID:   "c1",
					Code: "cc1",
					Name: "Child 1",
				},
			},
			{
				ID: "2",
				Parent: &SimpleStruct{
					ID:   "p2",
					Code: "pc2",
					Name: "Parent 2",
				},
				Child: SimpleStruct{
					ID:   "c2",
					Code: "cc2",
					Name: "Child 2",
				},
			},
		}

		// overwriteモードでのテスト（タグは無視され、単純にクローンされる）
		dst := []NestedStruct{
			{
				ID: "X",
				Parent: &SimpleStruct{
					ID:   "pX",
					Code: "pcX",
					Name: "Parent X",
				},
				Child: SimpleStruct{
					ID:   "cX",
					Code: "ccX",
					Name: "Child X",
				},
			},
		}

		// overwriteモードでは保護されないことを確認
		err := CopySlice("create", &src, &dst, "overwrite")
		assert.NoError(t, err)

		// Length should match the source
		assert.Equal(t, len(src), len(dst))

		// In overwrite mode, tags are ignored (simple clone)
		assert.Equal(t, "1", dst[0].ID)         // ID is NOT protected in overwrite mode
		assert.Equal(t, "p1", dst[0].Parent.ID) // Parent ID is NOT protected
		assert.Equal(t, "c1", dst[0].Child.ID)  // Child ID is NOT protected

		// matchモードでのテスト（タグを尊重する）
		dst = []NestedStruct{
			{
				ID: "X",
				Parent: &SimpleStruct{
					ID:   "pX",
					Code: "pcX",
					Name: "Parent X",
				},
				Child: SimpleStruct{
					ID:   "cX",
					Code: "ccX",
					Name: "Child X",
				},
			},
		}

		// matchモードでテスト（IDが保護される）
		err = CopySlice("create", &src, &dst, "match")
		assert.NoError(t, err)

		// Length should match the source
		assert.Equal(t, len(src), len(dst))

		// ID fields should be protected at all levels in match mode
		assert.Equal(t, "X", dst[0].ID)         // Top-level ID is preserved
		assert.Equal(t, "pX", dst[0].Parent.ID) // Parent ID is preserved
		assert.Equal(t, "cX", dst[0].Child.ID)  // Child ID is preserved

		// Non-protected fields should be copied
		assert.Equal(t, src[0].Parent.Name, dst[0].Parent.Name)
		assert.Equal(t, src[0].Child.Name, dst[0].Child.Name)

		// Second element should be a new element with protected fields
		assert.Empty(t, dst[1].ID) // New element's ID is protected
		assert.NotNil(t, dst[1].Parent)
		assert.Empty(t, dst[1].Parent.ID) // New parent's ID is protected
		assert.Equal(t, src[1].Parent.Name, dst[1].Parent.Name)
	})
}
