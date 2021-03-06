package helpers

import (
	"github.com/bsm/reason/classifiers"
	"github.com/bsm/reason/core"
	"github.com/bsm/reason/testdata"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("nominalCObserver", func() {
	var subject CObserver

	model := testdata.ClassificationModel()
	predictor := model.Predictor("outlook")
	target := model.Target()
	instances := testdata.ClassificationData()

	BeforeEach(func() {
		subject = NewNominalCObserver()
		for _, inst := range instances {
			subject.Observe(target.Value(inst), predictor.Value(inst), inst.GetInstanceWeight())
		}
	})

	It("should observe", func() {
		o := subject.(*nominalCObserver)
		Expect(o.postSplit).To(HaveLen(2))
		Expect(o.HeapSize()).To(Equal(56))
	})

	DescribeTable("should calculate probabilty",
		func(tv, pv string, p float64) {
			Expect(subject.Probability(
				target.ValueOf(tv),
				predictor.ValueOf(pv),
			)).To(BeNumerically("~", p, 0.001))
		},

		// 0.333 + 0.417 + 0.250 = 1.0
		Entry("play if sunny", "yes", "sunny", 0.333),
		Entry("play if overcast", "yes", "overcast", 0.417),
		Entry("play if rainy", "yes", "rainy", 0.250),

		// 0.375 + 0.125 + 0.500 = 1.0
		Entry("don't play if sunny", "no", "sunny", 0.375),
		Entry("don't play if overcast", "no", "overcast", 0.125),
		Entry("don't play if rainy", "no", "rainy", 0.500),
	)

	It("should calculate best split", func() {
		s := subject.BestSplit(
			classifiers.InfoGainSplitCriterion{MinBranchFrac: 0.1},
			predictor,
			[]float64{9.0, 5.0},
		)
		Expect(s.Merit()).To(BeNumerically("~", 0.247, 0.001))
		Expect(s.Range()).To(Equal(1.0))
		Expect(s.Condition()).To(BeAssignableToTypeOf(&nominalMultiwaySplitCondition{}))
		Expect(s.Condition().Predictor().Name).To(Equal("outlook"))

		postStats := s.PostStats()
		Expect(postStats).To(HaveLen(3))
		Expect(postStats[0].State()).To(Equal(core.Prediction{
			{Value: 0, Votes: 2},
			{Value: 1, Votes: 3},
		}))
		Expect(postStats[1].State()).To(Equal(core.Prediction{
			{Value: 0, Votes: 4},
			{Value: 1, Votes: 0},
		}))
		Expect(postStats[2].State()).To(Equal(core.Prediction{
			{Value: 0, Votes: 3},
			{Value: 1, Votes: 2},
		}))
	})

})

var _ = Describe("gaussianCObserver", func() {
	var subject CObserver

	predictor := &core.Attribute{Name: "len", Kind: core.AttributeKindNumeric}
	target := &core.Attribute{Name: "class", Kind: core.AttributeKindNominal}
	instances := []core.Instance{
		core.MapInstance{"len": 1.4, "class": "a"},
		core.MapInstance{"len": 1.3, "class": "a"},
		core.MapInstance{"len": 1.5, "class": "a"},
		core.MapInstance{"len": 4.1, "class": "b"},
		core.MapInstance{"len": 3.7, "class": "b"},
		core.MapInstance{"len": 4.9, "class": "b"},
		core.MapInstance{"len": 4.0, "class": "b"},
		core.MapInstance{"len": 3.3, "class": "b"},
		core.MapInstance{"len": 6.3, "class": "c"},
		core.MapInstance{"len": 5.8, "class": "c"},
		core.MapInstance{"len": 5.1, "class": "c"},
		core.MapInstance{"len": 5.3, "class": "c"},
	}

	BeforeEach(func() {
		subject = NewNumericCObserver(4)
		for _, inst := range instances {
			tv := target.Value(inst)
			pv := predictor.Value(inst)
			subject.Observe(tv, pv, inst.GetInstanceWeight())
		}
	})

	It("should observe", func() {
		o := subject.(*gaussianCObserver)
		Expect(o.minMax.Points(4)).To(Equal([]float64{2.3, 3.3, 4.3, 5.3}))
		Expect(o.postSplit).To(HaveLen(3))
		Expect(o.HeapSize()).To(Equal(240))
	})

	It("should not calculate probability", func() {
		Expect(subject.Probability(
			target.ValueOf("b"),
			predictor.ValueOf(4.5),
		)).To(BeNumerically("~", 0.47, 0.01))

		Expect(subject.Probability(
			target.ValueOf("a"),
			predictor.ValueOf(4.5),
		)).To(BeNumerically("~", 0.00, 0.01))

		Expect(subject.Probability(
			target.ValueOf("a"),
			predictor.ValueOf(1.7),
		)).To(BeNumerically("~", 0.04, 0.01))
	})

	It("should calculate best split", func() {
		s := subject.BestSplit(
			classifiers.InfoGainSplitCriterion{MinBranchFrac: 0.1},
			predictor,
			[]float64{3.0, 5.0, 4.0},
		)
		Expect(s.Merit()).To(BeNumerically("~", 0.811, 0.001))
		Expect(s.Range()).To(BeNumerically("~", 1.585, 0.001))
		Expect(s.Condition()).To(BeAssignableToTypeOf(&numericBinarySplitCondition{}))
		Expect(s.Condition().Predictor().Name).To(Equal("len"))
		Expect(s.Condition().(*numericBinarySplitCondition).splitValue).To(Equal(2.30))

		postStats := s.PostStats()
		Expect(postStats).To(HaveLen(2))
		Expect(postStats[0].State()).To(Equal(core.Prediction{
			{Value: 0, Votes: 3},
		}))
		Expect(postStats[1].State()).To(Equal(core.Prediction{
			{Value: 0, Votes: 0},
			{Value: 1, Votes: 5},
			{Value: 2, Votes: 4},
		}))
	})

})

var _ = Describe("nominalRObserver", func() {
	var subject RObserver
	var preSplit *core.NumSeries

	model := testdata.RegressionModel()
	predictor := model.Predictor("outlook")
	target := model.Target()
	instances := testdata.RegressionData()

	BeforeEach(func() {
		subject = NewNominalRObserver()
		preSplit = new(core.NumSeries)

		for _, inst := range instances {
			tv := target.Value(inst)
			pv := predictor.Value(inst)
			subject.Observe(tv, pv, inst.GetInstanceWeight())
			preSplit.Append(tv.Value(), inst.GetInstanceWeight())
		}
	})

	It("should observe", func() {
		o := subject.(*nominalRObserver)
		Expect(o.postSplit).To(HaveLen(3))
		Expect(o.postSplit[0].StdDev()).To(BeNumerically("~", 7.78, 0.01))
		Expect(o.postSplit[1].StdDev()).To(BeNumerically("~", 3.49, 0.01))
		Expect(o.postSplit[2].StdDev()).To(BeNumerically("~", 10.87, 0.01))
		Expect(o.HeapSize()).To(Equal(112))
	})

	It("should calculate best split", func() {
		s := subject.BestSplit(
			classifiers.VRSplitCriterion{},
			predictor,
			preSplit,
		)
		Expect(s.Merit()).To(BeNumerically("~", 19.572, 0.001))
		Expect(s.Range()).To(Equal(1.0))
		Expect(s.Condition()).To(BeAssignableToTypeOf(&nominalMultiwaySplitCondition{}))
		Expect(s.Condition().Predictor().Name).To(Equal("outlook"))
	})

})

var _ = Describe("gaussianRObserver", func() {
	var subject RObserver
	var preSplit *core.NumSeries

	predictor := &core.Attribute{Name: "area", Kind: core.AttributeKindNumeric}
	target := &core.Attribute{Name: "price", Kind: core.AttributeKindNumeric}
	instances := []core.MapInstance{
		{"area": 1.1, "price": 4.5},
		{"area": 1.2, "price": 4.5},
		{"area": 1.5, "price": 5.0},
		{"area": 0.9, "price": 3.8},
		{"area": 1.3, "price": 5.8},
		{"area": 1.5, "price": 5.6},
		{"area": 0.8, "price": 3.2},
		{"area": 2.6, "price": 8.2},
		{"area": 1.0, "price": 3.9},
		{"area": 1.6, "price": 5.1},
		{"area": 1.8, "price": 8.7},
		{"area": 1.6, "price": 6.0},
	}

	BeforeEach(func() {
		subject = NewNumericRObserver(5)
		preSplit = new(core.NumSeries)

		for _, inst := range instances {
			tv := target.Value(inst)
			pv := predictor.Value(inst)
			subject.Observe(tv, pv, inst.GetInstanceWeight())
			preSplit.Append(tv.Value(), inst.GetInstanceWeight())
		}
	})

	It("should observe", func() {
		o := subject.(*gaussianRObserver)
		Expect(o.minMax.SplitPoints(5)).To(Equal([]float64{1.1, 1.4, 1.7, 2, 2.3}))
		Expect(o.tuples).To(HaveLen(12))
		Expect(o.HeapSize()).To(Equal(368))
	})

	It("should calculate best split", func() {
		s := subject.BestSplit(
			classifiers.VRSplitCriterion{},
			predictor,
			preSplit,
		)
		Expect(s.Merit()).To(BeNumerically("~", 1.911, 0.001))
		Expect(s.Range()).To(Equal(1.0))
		Expect(s.Condition()).To(BeAssignableToTypeOf(&numericBinarySplitCondition{}))
		Expect(s.Condition().Predictor().Name).To(Equal("area"))
		Expect(s.Condition().(*numericBinarySplitCondition).splitValue).To(Equal(1.7))
	})

})
