/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Input,
  Select,
  Spin,
  Table,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconRefresh, IconSave, IconSearch } from '@douyinfe/semi-icons';
import { API } from '../../helpers';

const { Title, Text } = Typography;
const KEY_STORAGE = 'relayx_ops_read_key';

const formatDateTime = (value) => {
  if (!value) return '-';
  const normalized = typeof value === 'number' && value > 0 && value < 100000000000 ? value * 1000 : value;
  const date = new Date(normalized);
  if (Number.isNaN(date.getTime())) return String(value);
  const pad = (num) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
};

const formatNumber = (value) => {
  const num = Number(value || 0);
  return num.toLocaleString('zh-CN');
};

const formatPercent = (value) => `${(Number(value || 0) * 100).toFixed(2)}%`;

const asArray = (value) => (Array.isArray(value) ? value : []);

const buildAuthHeaders = (key) => ({ Authorization: `Bearer ${key}` });

const statusText = (value) => {
  if (value === true || value === 1 || value === '1' || value === 'enabled') {
    return '启用';
  }
  if (value === false || value === 0 || value === '0' || value === 'disabled') {
    return '禁用';
  }
  return value || '未知';
};

const StatCard = ({ title, value, desc, tone = 'default' }) => (
  <Card className={`relayx-stat-card relayx-stat-${tone}`} bodyStyle={{ padding: 14 }}>
    <Text type='tertiary' size='small'>{title}</Text>
    <div className='relayx-stat-value'>{value}</div>
    {desc && <Text type='quaternary' size='small'>{desc}</Text>}
  </Card>
);

const MiniList = ({ title, items, columns }) => (
  <Card className='relayx-panel-card' title={title} bodyStyle={{ padding: 0 }}>
    {items.length === 0 ? (
      <div className='relayx-empty'><Empty title='暂无数据' description='本地空库返回 0 属于正常情况' /></div>
    ) : (
      <Table
        size='small'
        pagination={false}
        dataSource={items}
        columns={columns}
        rowKey={(record, index) => record.id || record.user_id || record.channel_id || record.name || index}
      />
    )}
  </Card>
);

const RelayXOps = () => {
  const [opsKey, setOpsKey] = useState(() => localStorage.getItem(KEY_STORAGE) || '');
  const [draftKey, setDraftKey] = useState(() => localStorage.getItem(KEY_STORAGE) || '');
  const [windowValue, setWindowValue] = useState('7d');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [lastLoadedAt, setLastLoadedAt] = useState('');
  const [summary, setSummary] = useState(null);
  const [users, setUsers] = useState([]);
  const [logs, setLogs] = useState([]);
  const [models, setModels] = useState([]);
  const [channels, setChannels] = useState([]);
  const [billing, setBilling] = useState(null);
  const [securityEvents, setSecurityEvents] = useState([]);

  const effectiveKey = opsKey || draftKey;

  const saveKey = useCallback(() => {
    const next = draftKey.trim();
    setOpsKey(next);
    localStorage.setItem(KEY_STORAGE, next);
    Toast.success('只读 Key 已保存到本地浏览器');
  }, [draftKey]);

  const loadOpsData = useCallback(async () => {
    const key = (opsKey || draftKey).trim();
    if (!key) {
      setError('请先填写 RELAYX 只读 Key。该 Key 只应来自 RELAYX_OPS_READ_KEY 或 RelayXOpsReadKey。');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const config = {
        params: { window: windowValue },
        headers: buildAuthHeaders(key),
        skipErrorHandler: true,
        disableDuplicate: true,
      };
      const [summaryRes, usersRes, logsRes, modelsRes, channelsRes, billingRes, securityRes] = await Promise.all([
        API.get('/api/relayx/ops/summary', config),
        API.get('/api/relayx/ops/users', config),
        API.get('/api/relayx/ops/logs', config),
        API.get('/api/relayx/ops/models', config),
        API.get('/api/relayx/ops/channels', config),
        API.get('/api/relayx/ops/billing', config),
        API.get('/api/relayx/ops/security-events', config),
      ]);
      setSummary(summaryRes.data?.data || {});
      setUsers(asArray(usersRes.data?.data?.users?.top_by_usage || usersRes.data?.data?.users?.recent_users || usersRes.data?.data?.users || usersRes.data?.data));
      setLogs(asArray(logsRes.data?.data?.recent || logsRes.data?.data?.items || logsRes.data?.data?.logs || logsRes.data?.data));
      setModels(asArray(modelsRes.data?.data?.items || modelsRes.data?.data?.models || modelsRes.data?.data));
      setChannels(asArray(channelsRes.data?.data?.items || channelsRes.data?.data?.channels || channelsRes.data?.data));
      setBilling(billingRes.data?.data || {});
      setSecurityEvents(asArray(securityRes.data?.data?.events || securityRes.data?.data?.items || securityRes.data?.data));
      setLastLoadedAt(new Date().toISOString());
      setOpsKey(key);
      localStorage.setItem(KEY_STORAGE, key);
    } catch (err) {
      const message = err?.response?.data?.message || err?.response?.data?.error || err?.message || '加载失败';
      setError(`接口请求失败：${message}`);
    } finally {
      setLoading(false);
    }
  }, [draftKey, opsKey, windowValue]);

  useEffect(() => {
    if (effectiveKey) {
      loadOpsData();
    }
  }, [windowValue]);

  const summaryData = summary || {};
  const topUsers = asArray(summaryData.top_users || users).slice(0, 8);
  const topModels = asArray(summaryData.top_models || models).slice(0, 8);
  const topChannels = asArray(summaryData.top_channels || channels).slice(0, 8);

  const stats = useMemo(() => [
    {
      title: '用户总数',
      value: formatNumber(summaryData.total_users ?? summaryData.users?.total),
      desc: `活跃 ${formatNumber(summaryData.active_users ?? summaryData.users?.active)}`,
      tone: 'blue',
    },
    {
      title: '窗口请求',
      value: formatNumber(summaryData.total_requests ?? summaryData.requests?.total ?? summaryData.requests?.requests),
      desc: `成功 ${formatNumber(summaryData.success_requests ?? summaryData.requests?.success)}`,
      tone: 'green',
    },
    {
      title: '失败率',
      value: formatPercent(summaryData.failure_rate ?? summaryData.requests?.failure_rate),
      desc: `失败 ${formatNumber(summaryData.failed_requests ?? summaryData.requests?.failed ?? summaryData.requests?.failures)}`,
      tone: 'orange',
    },
    {
      title: '窗口消耗',
      value: formatNumber(summaryData.quota ?? summaryData.used_quota ?? summaryData.billing?.used_quota),
      desc: billing?.currency ? `币种 ${billing.currency}` : 'quota 聚合值',
      tone: 'purple',
    },
  ], [billing?.currency, summaryData]);

  const userColumns = [
    { title: '用户ID', dataIndex: 'user_id', width: 90, render: (v, r) => v ?? r.id ?? '-' },
    { title: '标识', dataIndex: 'username', render: (v, r) => v || r.email || r.name || '-' },
    { title: '请求', dataIndex: 'requests', width: 90, render: formatNumber },
    { title: '消耗', dataIndex: 'quota', width: 90, render: formatNumber },
  ];

  const modelColumns = [
    { title: '模型', dataIndex: 'model_name', render: (v, r) => v || r.name || '-' },
    { title: '请求', dataIndex: 'requests', width: 90, render: formatNumber },
    { title: '失败率', dataIndex: 'failure_rate', width: 90, render: formatPercent },
  ];

  const channelColumns = [
    { title: '渠道', dataIndex: 'channel_id', width: 90, render: (v, r) => v ?? r.id ?? '-' },
    { title: '名称', dataIndex: 'name', render: (v) => v || '已脱敏' },
    { title: '状态', dataIndex: 'status', width: 90, render: (v) => <Tag size='small'>{statusText(v)}</Tag> },
    { title: '失败率', dataIndex: 'failure_rate', width: 90, render: formatPercent },
  ];

  const logColumns = [
    { title: '时间', dataIndex: 'created_at', width: 170, render: formatDateTime },
    { title: '类型', dataIndex: 'type', width: 110, render: (v) => <Tag size='small'>{v || '日志'}</Tag> },
    { title: '摘要', dataIndex: 'summary', render: (v, r) => v || r.message || r.reason || '已隐藏原始内容' },
  ];

  return (
    <div className='relayx-ops-page'>
      <div className='relayx-ops-header'>
        <div>
          <Title heading={3} style={{ margin: 0 }}>RELAYX 运营概览</Title>
          <Text type='secondary'>只读巡检面板 · 本地空库显示 0 属于正常 · 不展示密钥/Prompt/响应原文</Text>
        </div>
        <div className='relayx-ops-actions'>
          <Select value={windowValue} onChange={setWindowValue} style={{ width: 110 }}>
            <Select.Option value='1d'>近 1 天</Select.Option>
            <Select.Option value='7d'>近 7 天</Select.Option>
            <Select.Option value='30d'>近 30 天</Select.Option>
          </Select>
          <Button icon={<IconRefresh />} loading={loading} onClick={loadOpsData}>刷新</Button>
        </div>
      </div>

      <Card className='relayx-key-card' bodyStyle={{ padding: 14 }}>
        <div className='relayx-key-row'>
          <Input
            prefix={<IconSearch />}
            value={draftKey}
            onChange={setDraftKey}
            placeholder='请输入 RELAYX 运营只读 Key（RELAYX_OPS_READ_KEY / RelayXOpsReadKey）'
            onEnterPress={() => {
              saveKey();
              loadOpsData();
            }}
          />
          <Button type='primary' icon={<IconSave />} onClick={saveKey}>保存 Key</Button>
          <Button loading={loading} onClick={loadOpsData}>加载数据</Button>
        </div>
        <Text type='quaternary' size='small'>更新时间：{lastLoadedAt ? formatDateTime(lastLoadedAt) : '尚未加载'}</Text>
      </Card>

      {error && <Banner type='danger' description={error} closeIcon={null} />}

      <Spin spinning={loading}>
        <div className='relayx-stats-grid'>
          {stats.map((item) => <StatCard key={item.title} {...item} />)}
        </div>

        <div className='relayx-main-grid'>
          <MiniList title='Top 用户' items={topUsers} columns={userColumns} />
          <MiniList title='Top 模型' items={topModels} columns={modelColumns} />
          <MiniList title='Top 渠道' items={topChannels} columns={channelColumns} />
        </div>

        <div className='relayx-bottom-grid'>
          <MiniList title='最近日志 / 失败摘要' items={logs.slice(0, 8)} columns={logColumns} />
          <MiniList title='安全事件' items={securityEvents.slice(0, 8)} columns={logColumns} />
        </div>
      </Spin>
    </div>
  );
};

export default RelayXOps;
