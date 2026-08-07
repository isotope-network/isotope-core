package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestCosineSimilarity_Identical проверяет, что косинус одинаковых векторов = 1
func TestCosineSimilarity_Identical(t *testing.T) {
	vec := make([]float64, VectorDim)
	for i := range vec {
		vec[i] = 0.1 * float64(i+1)
	}
	result := cosineSimilarity(vec, vec)
	if result < 0.999 || result > 1.001 {
		t.Errorf("cosineSimilarity(identical) = %f, want ~1.0", result)
	}
}

// TestCosineSimilarity_Orthogonal проверяет ортогональные векторы
func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	result := cosineSimilarity(a, b)
	if result > 0.001 {
		t.Errorf("cosineSimilarity(orthogonal) = %f, want ~0", result)
	}
}

// TestCosineSimilarity_ZeroVector проверяет защиту от деления на ноль
func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float64{0, 0, 0}
	b := []float64{1, 2, 3}
	result := cosineSimilarity(a, b)
	if result != 0 {
		t.Errorf("cosineSimilarity(zero, normal) = %f, want 0", result)
	}
}

// TestCosineSimilarity_Negative проверяет противоположные векторы
func TestCosineSimilarity_Negative(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{-1, -2, -3}
	result := cosineSimilarity(a, b)
	if result > -0.999 || result < -1.001 {
		t.Errorf("cosineSimilarity(opposite) = %f, want ~-1.0", result)
	}
}

// TestCosineSimilarity_Symmetric проверяет симметричность
func TestCosineSimilarity_Symmetric(t *testing.T) {
	a := []float64{0.5, 0.3, 0.8}
	b := []float64{0.2, 0.9, 0.1}
	r1 := cosineSimilarity(a, b)
	r2 := cosineSimilarity(b, a)
	if r1 != r2 {
		t.Errorf("cosineSimilarity not symmetric: %f vs %f", r1, r2)
	}
}

// TestCosineSimilarity_Empty проверяет пустые векторы
func TestCosineSimilarity_Empty(t *testing.T) {
	result := cosineSimilarity([]float64{}, []float64{})
	if result != 0 {
		t.Errorf("cosineSimilarity(empty, empty) = %f, want 0", result)
	}
}

// TestCosineSimilarity_DifferentLength проверяет векторы разной длины
func TestCosineSimilarity_DifferentLength(t *testing.T) {
	result := cosineSimilarity([]float64{1, 2}, []float64{1, 2, 3})
	if result != 0 {
		t.Errorf("cosineSimilarity(different length) = %f, want 0", result)
	}
}

// TestCosineSimilarity_NotNaN проверяет отсутствие NaN в результате
func TestCosineSimilarity_NotNaN(t *testing.T) {
	a := []float64{1e-300, 1e-300}
	b := []float64{1e-300, 1e-300}
	result := cosineSimilarity(a, b)
	if result != result { // NaN check: NaN != NaN
		t.Error("cosineSimilarity returned NaN")
	}
}

// TestTextToVector_Deterministic проверяет, что одинаковый текст даёт одинаковый вектор
func TestTextToVector_Deterministic(t *testing.T) {
	v1 := textToVector("привет")
	v2 := textToVector("привет")
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("textToVector not deterministic at index %d: %f vs %f", i, v1[i], v2[i])
			return
		}
	}
}

// TestTextToVector_DifferentTexts проверяет, что разные тексты дают разные векторы
func TestTextToVector_DifferentTexts(t *testing.T) {
	v1 := textToVector("мир")
	v2 := textToVector("война")
	same := true
	for i := range v1 {
		if v1[i] != v2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("textToVector returned same vector for different texts")
	}
}

// TestTextToVector_Dimension проверяет размерность вектора
func TestTextToVector_Dimension(t *testing.T) {
	v := textToVector("тест")
	if len(v) != VectorDim {
		t.Errorf("textToVector dimension = %d, want %d", len(v), VectorDim)
	}
}

// TestTextToVector_Range проверяет, что значения в [0, 1]
func TestTextToVector_Range(t *testing.T) {
	v := textToVector("тестовое сообщение достаточно длинное для проверки")
	for i, val := range v {
		if val < 0 || val > 1.0 {
			t.Errorf("textToVector[%d] = %f, out of [0, 1]", i, val)
		}
	}
}

// TestTextToVector_EmptyString проверяет пустую строку
func TestTextToVector_EmptyString(t *testing.T) {
	v := textToVector("")
	for _, val := range v {
		if val != 0 {
			t.Errorf("textToVector(empty) has non-zero value: %f", val)
			return
		}
	}
}

// TestTextToVector_English проверяет, что английский текст тоже векторизуется
func TestTextToVector_English(t *testing.T) {
	v := textToVector("hello world")
	hasValue := false
	for _, val := range v {
		if val > 0 {
			hasValue = true
			break
		}
	}
	if !hasValue {
		t.Error("textToVector(english) returned zero vector")
	}
}

// TestVectorToText_NotPanic проверяет, что vectorToText не паникует
func TestVectorToText_NotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("vectorToText panicked: %v", r)
		}
	}()

	// Пустой вектор
	_ = vectorToText([]float64{})

	// Нормальный вектор
	vec := make([]float64, VectorDim)
	for i := range vec {
		vec[i] = 0.5
	}
	_ = vectorToText(vec)

	// Экстремальные значения
	extreme := make([]float64, VectorDim)
	for i := range extreme {
		extreme[i] = 1.0
	}
	_ = vectorToText(extreme)

	// Отрицательные значения
	neg := make([]float64, VectorDim)
	for i := range neg {
		neg[i] = -1.0
	}
	_ = vectorToText(neg)
}

// TestVectorToText_Russian проверяет, что результат содержит русские буквы или пробелы
func TestVectorToText_Russian(t *testing.T) {
	vec := make([]float64, VectorDim)
	for i := range vec {
		vec[i] = float64(i) / float64(VectorDim)
	}
	result := vectorToText(vec)
	if len(result) < 3 {
		t.Errorf("vectorToText result too short: %q", result)
	}
	// Проверяем, что есть хотя бы одна русская буква или пробел
	hasRussianOrSpace := false
	for _, ch := range result {
		if (ch >= 'а' && ch <= 'я') || (ch >= 'А' && ch <= 'Я') || ch == ' ' || ch == ',' || ch == '.' || ch == 'ё' || ch == 'Ё' {
			hasRussianOrSpace = true
			break
		}
	}
	if !hasRussianOrSpace {
		t.Errorf("vectorToText result has no Russian letters: %q", result)
	}
}

// TestVectorToText_Length проверяет длину ответа в допустимом диапазоне
func TestVectorToText_Length(t *testing.T) {
	vecLow := make([]float64, VectorDim)
	for i := range vecLow {
		vecLow[i] = 0.01
	}
	resultLow := vectorToText(vecLow)
	if utf8.RuneCountInString(resultLow) < 3 {
		t.Errorf("vectorToText min length violated for low values: runes=%d, text=%q", utf8.RuneCountInString(resultLow), resultLow)
	}

	vecHigh := make([]float64, VectorDim)
	for i := range vecHigh {
		vecHigh[i] = 1.0
	}
	resultHigh := vectorToText(vecHigh)
	if utf8.RuneCountInString(resultHigh) < 3 {
		t.Errorf("vectorToText min length violated for high values: runes=%d, text=%q", utf8.RuneCountInString(resultHigh), resultHigh)
	}
	if utf8.RuneCountInString(resultHigh) > 30 {
		t.Errorf("vectorToText max length violated: runes=%d, text=%q", utf8.RuneCountInString(resultHigh), resultHigh)
	}
}

// TestVectorToText_Deterministic проверяет детерминизм
func TestVectorToText_Deterministic(t *testing.T) {
	vec := make([]float64, VectorDim)
	for i := range vec {
		vec[i] = float64(i*7+3) / float64(VectorDim*7)
	}
	r1 := vectorToText(vec)
	r2 := vectorToText(vec)
	if r1 != r2 {
		t.Errorf("vectorToText not deterministic: %q vs %q", r1, r2)
	}
}

// TestVectorToText_BigramsCoverage проверяет, что все русские буквы в таблице переходов имеют записи
func TestVectorToText_BigramsCoverage(t *testing.T) {
	russianLetters := []rune{
		'а', 'б', 'в', 'г', 'д', 'е', 'ё', 'ж', 'з', 'и', 'й', 'к', 'л', 'м',
		'н', 'о', 'п', 'р', 'с', 'т', 'у', 'ф', 'х', 'ц', 'ч', 'ш', 'щ', 'ъ',
		'ы', 'ь', 'э', 'ю', 'я', ' ', ',', '.',
	}
	for _, ch := range russianLetters {
		if _, exists := bigramTransitions[ch]; !exists {
			t.Errorf("bigramTransitions missing entry for rune %c", ch)
		}
	}
}

// TestVectorToText_StartersCoverage проверяет, что starters содержит только буквы из таблицы
func TestVectorToText_StartersCoverage(t *testing.T) {
	for _, s := range starters {
		if _, exists := bigramTransitions[s.char]; !exists {
			t.Errorf("starter %c has no bigramTransitions entry", s.char)
		}
	}
}

// TestHashToVector проверяет хеш-функцию
func TestHashToVector_Dimension(t *testing.T) {
	v := hashToVector("abc123")
	if len(v) != VectorDim {
		t.Errorf("hashToVector dimension = %d, want %d", len(v), VectorDim)
	}
}

// TestHashToVector_SameHashSameVector проверяет детерминизм
func TestHashToVector_SameHashSameVector(t *testing.T) {
	v1 := hashToVector("test")
	v2 := hashToVector("test")
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("hashToVector not deterministic at index %d", i)
			return
		}
	}
}

// TestSqrt проверяет квадратный корень
func TestSqrt_Positive(t *testing.T) {
	result := sqrt(4.0)
	if result < 1.999 || result > 2.001 {
		t.Errorf("sqrt(4) = %f, want ~2", result)
	}
}

func TestSqrt_Zero(t *testing.T) {
	if sqrt(0) != 0 {
		t.Errorf("sqrt(0) = %f, want 0", sqrt(0))
	}
}

func TestSqrt_Negative(t *testing.T) {
	if sqrt(-1) != 0 {
		t.Errorf("sqrt(-1) = %f, want 0", sqrt(-1))
	}
}

// TestGenerateMsgID проверяет генерацию ID
func TestGenerateMsgID_NotEmpty(t *testing.T) {
	id := generateMsgID("тест")
	if len(id) == 0 {
		t.Error("generateMsgID returned empty string")
	}
}

func TestGenerateMsgID_HexChars(t *testing.T) {
	id := generateMsgID("hello")
	for _, ch := range id {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("generateMsgID contains non-hex character: %c in %s", ch, id)
			return
		}
	}
}

// TestHashText проверяет хеширование текста
func TestHashText_NotEmpty(t *testing.T) {
	h := hashText("something")
	if len(h) == 0 {
		t.Error("hashText returned empty string")
	}
}

// TestForward проверяет прямой проход нейросети
func TestForward_Dimensions(t *testing.T) {
	input := make([]float64, VectorDim)
	for i := range input {
		input[i] = 0.5
	}
	layers := [][]float64{make([]float64, VectorDim)}
	for i := range layers[0] {
		layers[0][i] = 1.0
	}
	output, current := forward(input, layers)
	if len(output) != VectorDim {
		t.Errorf("forward output dimension = %d, want %d", len(output), VectorDim)
	}
	if len(current) != VectorDim {
		t.Errorf("forward current dimension = %d, want %d", len(current), VectorDim)
	}
}

// TestForward_Range проверяет, что выходные значения в [-1, 1]
func TestForward_Range(t *testing.T) {
	input := make([]float64, VectorDim)
	for i := range input {
		input[i] = 0.7
	}
	layers := [][]float64{
		make([]float64, VectorDim),
		make([]float64, VectorDim),
	}
	for i := range layers[0] {
		layers[0][i] = 0.5
		layers[1][i] = 1.5
	}
	output, _ := forward(input, layers)
	for i, val := range output {
		if val < -1.0 || val > 1.0 {
			t.Errorf("forward output[%d] = %f, out of [-1, 1]", i, val)
		}
	}
}

// TestTrain_Dimensions проверяет, что train не меняет размерность
func TestTrain_Dimensions(t *testing.T) {
	input := make([]float64, VectorDim)
	output := make([]float64, VectorDim)
	target := make([]float64, VectorDim)
	layers := [][]float64{make([]float64, VectorDim)}

	result := train(layers, input, output, target, 0.01)
	if len(result) != len(layers) {
		t.Errorf("train changed number of layers: %d -> %d", len(layers), len(result))
	}
	if len(result[0]) != VectorDim {
		t.Errorf("train changed layer dimension: %d -> %d", VectorDim, len(result[0]))
	}
}

// TestTrain_Range проверяет, что веса остаются в [-1, 1]
func TestTrain_Range(t *testing.T) {
	input := make([]float64, VectorDim)
	output := make([]float64, VectorDim)
	target := make([]float64, VectorDim)
	for i := range target {
		target[i] = 1.0
	}
	layers := [][]float64{make([]float64, VectorDim)}
	for i := range layers[0] {
		layers[0][i] = 0.5
	}

	// Много итераций обучения
	for iter := 0; iter < 100; iter++ {
		layers = train(layers, input, output, target, 0.1)
	}
	for i, val := range layers[0] {
		if val < -1.0 || val > 1.0 {
			t.Errorf("train weight[%d] = %f, out of [-1, 1]", i, val)
		}
	}
}

// TestTextToVector_Bigrams проверяет, что биграммы дают вклад в вектор
func TestTextToVector_Bigrams(t *testing.T) {
	v1 := textToVector("абв")
	v2 := textToVector("авб")
	different := false
	for i := range v1 {
		if v1[i] != v2[i] {
			different = true
			break
		}
	}
	if !different {
		t.Error("textToVector should produce different vectors for different bigrams")
	}
}

// TestTextToVector_SingleChar проверяет одиночный символ
func TestTextToVector_SingleChar(t *testing.T) {
	v := textToVector("я")
	if len(v) != VectorDim {
		t.Errorf("textToVector(single char) dimension = %d, want %d", len(v), VectorDim)
	}
}

// TestTextToVector_VeryLong проверяет очень длинный текст
func TestTextToVector_VeryLong(t *testing.T) {
	longText := strings.Repeat("тест ", 1000)
	v := textToVector(longText)
	for i, val := range v {
		if val < 0 || val > 1.0 {
			t.Errorf("textToVector(long)[%d] = %f, out of [0, 1]", i, val)
		}
	}
}