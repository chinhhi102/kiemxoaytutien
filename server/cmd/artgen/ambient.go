package main

// Lớp "sự sống bản đồ" — đom đóm, bướm, chim, gà, lồng đèn, lá rơi, bóng mây.
//
// Bảy thứ này được `client/src/render/catalog.ts` khai báo và `ambient.ts` vẽ, nhưng artgen
// CHƯA BAO GIỜ có job nào sinh ra chúng. Cùng một lỗ hổng với `bamboo_wall` và
// `decals/cart_tracks`: danh mục bên client liệt kê, bên sinh ảnh không biết, và không có gì
// báo — lớp sinh động chỉ lặng lẽ không xuất hiện.
//
// Chúng KHÔNG chặn build (thiếu thì game bỏ qua, không vẽ placeholder), nên càng dễ nằm im
// lâu. Bản đồ 12 000×12 000 mà không một con bướm nào bay thì nhìn chết cứng.

// ambientArt mô tả từng thứ. Cỡ 128 px: chúng vẽ ra màn hình chỉ vài chục pixel, và đo hôm
// 30/10 cho thấy ảnh 256 cho vật nhỏ là thừa 4–5 lần (xem ghi chú ở atlas.go).
var ambientArt = []struct{ id, desc string }{
	{"falling_leaf", "a single small falling leaf seen from above, warm yellow-green, slight curl"},
	{"firefly", "a tiny glowing firefly, soft warm yellow halo around a small bright core"},
	{"butterfly", "a small butterfly seen from above with wings spread, pale blue and cream"},
	{"chicken", "a small plump chicken seen from above, brown and white feathers, walking"},
	{"lantern", "a small round paper lantern glowing warm orange, hanging"},
	{"bird", "a small bird seen from above with wings spread in flight, dark silhouette with soft edges"},
	// Bóng mây là thứ DUY NHẤT trong nhóm không có hình thù rõ: nó là một mảng tối mềm trôi
	// ngang bản đồ. Vẽ nó thành "đám mây" là sai — cái cần là cái BÓNG đổ xuống đất.
	{"cloud_shadow", "a soft irregular dark shadow blob with very blurred edges, semi transparent, as cast by a cloud onto the ground"},
}

func ambientJobs() []Job {
	jobs := make([]Job, 0, len(ambientArt))
	for _, a := range ambientArt {
		jobs = append(jobs, Job{
			Name: "ambient_" + a.id, Priority: 5, Mode: modeSprite,
			Out: "ambient/" + a.id + ".png", W: 128, H: 128, Pivot: "center",
			Prompt: styleObject + " " + a.desc + ", top-down view, small game decoration sprite" + negative,
		})
	}
	return jobs
}
