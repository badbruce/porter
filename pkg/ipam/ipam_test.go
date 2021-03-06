package ipam_test

import (
	"net"

	"github.com/kubesphere/porter/pkg/constant"

	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/ipam"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	testIPNet    *net.IPNet
	testIPNetStr string
)

var _ = Describe("Ipam", func() {
	BeforeEach(func() {
		testIPNetStr = "192.168.1.0/24"
		_, testIPNet, _ = net.ParseCIDR(testIPNetStr)
	})

	It("Should be ok with single ip", func() {
		ds := ipam.NewDataStore(ctrl.Log.WithName("setup"), nil)
		singleIP := "1.1.1.1"
		Expect(ds.AddEIPPool(singleIP, "singleEIP", false, constant.PorterProtocolBGP)).ShouldNot(HaveOccurred())
		Expect(*ds.GetEIPStatus(singleIP)).To(Equal(ipam.EIPStatus{
			Exist: true,
			EIPRef: &ipam.EIPRef{
				EIPRefName: "singleEIP",
				Address:    singleIP,
			},
		}))
		ip, err := ds.AssignIP("testSVC", "test", constant.PorterProtocolBGP)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ip.Address).To(Equal(singleIP))
		Expect(ds.UnassignIP(ip.Address)).ShouldNot(HaveOccurred())
		Expect(ds.RemoveEIPPool(singleIP, "singleEIP")).ShouldNot(HaveOccurred())
	})

	It("Should be ok to add eip", func() {
		ds := ipam.NewDataStore(ctrl.Log.WithName("setup"), nil)

		Expect(ds.AddEIPPool(testIPNetStr, "defaultPool", false, constant.PorterProtocolBGP)).ShouldNot(HaveOccurred())

		Expect(ds.GetEIPStatus("192.168.2.1").Exist).To(BeFalse())
		Expect(*ds.GetEIPStatus("192.168.1.1")).To(Equal(ipam.EIPStatus{
			Exist: true,
			EIPRef: &ipam.EIPRef{
				EIPRefName: "defaultPool",
				Address:    "192.168.1.1",
			},
		}))

		ip, err := ds.AssignIP("testSVC", "test", constant.PorterProtocolBGP)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(testIPNet.Contains(net.ParseIP(ip.Address))).To(BeTrue())
		Expect(*ds.GetEIPStatus(ip.Address)).To(Equal(ipam.EIPStatus{
			Exist: true,
			EIPRef: &ipam.EIPRef{
				EIPRefName: ip.EIPRefName,
				Address:    ip.Address,
				Service: types.NamespacedName{
					Namespace: "test",
					Name:      "testSVC",
				},
			},
			Used: true,
		}))
		_, err = ds.AssignSpecifyIP(ip.Address, constant.PorterProtocolBGP, "testSvc1", "default")
		Expect(errors.ReasonForError(err)).Should(Equal(errors.EIPIsUsedError))

		_, err = ds.AssignSpecifyIP("192.168.1.2", constant.PorterProtocolBGP, "testSvc1", "default")
		Expect(err).ShouldNot(HaveOccurred())

		_, err = ds.AssignSpecifyIP("192.168.1.0", constant.PorterProtocolBGP, "testSvc1", "default")
		Expect(errors.ReasonForError(err)).Should(Equal(errors.EIPNotExist))

		Expect(errors.ReasonForError(ds.RemoveEIPPool(testIPNetStr, "defaultPool"))).Should(Equal(errors.EIPIsUsedError))
		Expect(ds.UnassignIP(ip.Address)).ShouldNot(HaveOccurred())
		Expect(ds.UnassignIP("192.168.1.2")).ShouldNot(HaveOccurred())
		Expect(ds.RemoveEIPPool(testIPNetStr, "defaultPool")).ShouldNot(HaveOccurred())

		Expect(ds.AddEIPPool("192.168.98.1", "oneeip", false, constant.PorterProtocolBGP)).ShouldNot(HaveOccurred())

		ip, err = ds.AssignIP("testSVC", "test", constant.PorterProtocolBGP)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ip.Address).To(Equal("192.168.98.1"))

		_, err = ds.AssignIP("testSVC", "test", constant.PorterProtocolBGP)
		Expect(errors.ReasonForError(err)).Should(Equal(errors.EIPNotEnoughError))
	})

	It("Should be ok to add two eip pool in the meantime", func() {
		ds := ipam.NewDataStore(ctrl.Log.WithName("setup"), nil)
		Expect(ds.AddEIPPool(testIPNetStr, "defaultPool", false, constant.PorterProtocolBGP)).ShouldNot(HaveOccurred())
		Expect(ds.AddEIPPool("192.168.2.2", "defaultPool1", false, constant.PorterProtocolBGP)).ShouldNot(HaveOccurred())
		resp, err := ds.AssignIP("testSvc1", "default", constant.PorterProtocolBGP)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ds.UnassignIP(resp.Address)).ShouldNot(HaveOccurred())
		Expect(ds.RemoveEIPPool(testIPNetStr, "defaultPool")).ShouldNot(HaveOccurred())
		Expect(ds.RemoveEIPPool("192.168.2.2", "defaultPool1")).ShouldNot(HaveOccurred())
	})
})
