package consistence

import (
	"github.com/absolute8511/nsq/nsqd"
	"net"
	"net/rpc"
	"time"
)

const (
	RPC_TIMEOUT       = time.Duration(time.Second * 10)
	RPC_TIMEOUT_SHORT = time.Duration(time.Second)
)

type NsqdRpcClient struct {
	remote     string
	timeout    time.Duration
	connection *rpc.Client
}

func convertRpcError(err error, coordErr *CoordErr) *CoordErr {
	if err != nil {
		return NewCoordErr(err.Error(), CoordNetErr)
	}
	if coordErr != nil && coordErr.HasError() {
		return coordErr
	}
	return nil
}

func NewNsqdRpcClient(addr string, timeout time.Duration) (*NsqdRpcClient, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	return &NsqdRpcClient{
		remote:     addr,
		timeout:    timeout,
		connection: rpc.NewClient(conn),
	}, nil
}

func (self *NsqdRpcClient) Reconnect() error {
	conn, err := net.DialTimeout("tcp", self.remote, self.timeout)
	if err != nil {
		return err
	}
	self.connection.Close()
	self.connection = rpc.NewClient(conn)
	return nil
}

func (self *NsqdRpcClient) CallWithRetry(method string, arg interface{}, reply interface{}) error {
	for {
		err := self.connection.Call(method, arg, reply)
		if err == rpc.ErrShutdown {
			coordLog.Infof("rpc connection closed, error: %v", err)
			err = self.Reconnect()
			if err != nil {
				return err
			}
		} else {
			if err != nil {
				coordLog.Infof("rpc call %v error: %v", method, err)
			}
			return err
		}
	}
}

func (self *NsqdRpcClient) NotifyTopicLeaderSession(epoch int, topicInfo *TopicPartionMetaInfo, leaderSession *TopicLeaderSession) *CoordErr {
	var rpcInfo RpcTopicLeaderSession
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicLeaderSession = leaderSession.Session
	rpcInfo.TopicLeaderEpoch = leaderSession.LeaderEpoch
	rpcInfo.LeaderNode = leaderSession.LeaderNode
	rpcInfo.TopicName = topicInfo.Name
	rpcInfo.TopicPartition = topicInfo.Partition
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.NotifyTopicLeaderSession", rpcInfo, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) UpdateTopicInfo(epoch int, topicInfo *TopicPartionMetaInfo) *CoordErr {
	var rpcInfo RpcAdminTopicInfo
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicPartionMetaInfo = *topicInfo
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.UpdateTopicInfo", rpcInfo, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) EnableTopicWrite(epoch int, topicInfo *TopicPartionMetaInfo) *CoordErr {
	var rpcInfo RpcAdminTopicInfo
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicPartionMetaInfo = *topicInfo
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.EnableTopicWrite", rpcInfo, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) DisableTopicWrite(epoch int, topicInfo *TopicPartionMetaInfo) *CoordErr {
	var rpcInfo RpcAdminTopicInfo
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicPartionMetaInfo = *topicInfo
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.DisableTopicWrite", rpcInfo, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) GetTopicStats(topic string) (*NodeTopicStats, error) {
	var stat NodeTopicStats
	err := self.CallWithRetry("NsqdCoordRpcServer.GetTopicStats", topic, &stat)
	return &stat, err
}

func (self *NsqdRpcClient) UpdateCatchupForTopic(epoch int, info *TopicPartionMetaInfo) *CoordErr {
	var rpcReq RpcAdminTopicInfo
	rpcReq.TopicPartionMetaInfo = *info
	rpcReq.LookupdEpoch = epoch
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.UpdateCatchupForTopic", rpcReq, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) UpdateChannelsForTopic(epoch int, info *TopicPartionMetaInfo) *CoordErr {
	var rpcReq RpcAdminTopicInfo
	rpcReq.TopicPartionMetaInfo = *info
	rpcReq.LookupdEpoch = epoch
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.UpdateChannelsForTopic", rpcReq, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) UpdateChannelOffset(leaderSession *TopicLeaderSession, info *TopicPartionMetaInfo, channel string, offset ChannelConsumerOffset) *CoordErr {
	var updateInfo RpcChannelOffsetArg
	updateInfo.TopicName = info.Name
	updateInfo.TopicPartition = info.Partition
	updateInfo.TopicEpoch = info.Epoch
	updateInfo.TopicLeaderEpoch = leaderSession.LeaderEpoch
	updateInfo.TopicLeaderSession = leaderSession.Session
	updateInfo.Channel = channel
	updateInfo.ChannelOffset = offset
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.UpdateChannelOffset", updateInfo, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) PutMessage(leaderSession *TopicLeaderSession, info *TopicPartionMetaInfo, log CommitLogData, message *nsqd.Message) *CoordErr {
	var putData RpcPutMessage
	putData.LogData = log
	putData.TopicName = info.Name
	putData.TopicPartition = info.Partition
	putData.TopicMessage = message
	putData.TopicEpoch = info.Epoch
	putData.TopicLeaderEpoch = leaderSession.LeaderEpoch
	putData.TopicLeaderSession = leaderSession.Session
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.PutMessage", putData, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) PutMessages(leaderSession *TopicLeaderSession, info *TopicPartionMetaInfo, loglist []CommitLogData, messages []*nsqd.Message) *CoordErr {
	var putData RpcPutMessages
	putData.LogList = loglist
	putData.TopicName = info.Name
	putData.TopicPartition = info.Partition
	putData.TopicMessages = messages
	putData.TopicEpoch = info.Epoch
	putData.TopicLeaderEpoch = leaderSession.LeaderEpoch
	putData.TopicLeaderSession = leaderSession.Session
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordRpcServer.PutMessages", putData, &retErr)
	return convertRpcError(err, &retErr)
}

func (self *NsqdRpcClient) GetLastCommmitLogID(topicInfo *TopicPartionMetaInfo) (int64, error) {
	var req RpcCommitLogReq
	req.TopicName = topicInfo.Name
	req.TopicPartition = topicInfo.Partition
	var ret int64
	err := self.CallWithRetry("NsqdCoordRpcServer.GetLastCommitLogID", req, &ret)
	return ret, err
}

func (self *NsqdRpcClient) GetCommmitLogFromOffset(topicInfo *TopicPartionMetaInfo, offset int64) (int64, CommitLogData, error) {
	var req RpcCommitLogReq
	req.LogOffset = offset
	req.TopicName = topicInfo.Name
	req.TopicPartition = topicInfo.Partition
	var rsp RpcCommitLogRsp
	err := self.CallWithRetry("NsqdCoordRpcServer.GetCommmitLogFromOffset", req, &rsp)
	return rsp.LogOffset, rsp.LogData, err
}

func (self *NsqdRpcClient) PullCommitLogsAndData(topic string, partition int,
	startOffset int64, num int) ([]CommitLogData, [][]byte, error) {
	var r RpcPullCommitLogsReq
	r.TopicName = topic
	r.TopicPartition = partition
	r.StartLogOffset = startOffset
	r.LogMaxNum = num
	var ret RpcPullCommitLogsRsp
	err := self.CallWithRetry("NsqdCoordRpcServer.PullCommitLogs", r, &ret)
	return ret.Logs, ret.DataList, err
}

func (self *NsqdRpcClient) CallRpcTest(data string) (string, *CoordErr) {
	var req RpcTestReq
	req.Data = data
	var ret RpcTestRsp
	err := self.CallWithRetry("NsqdCoordRpcServer.TestRpcError", req, &ret)
	return ret.RspData, convertRpcError(err, ret.RetErr)
}