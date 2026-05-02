import json
from http.server import HTTPServer, BaseHTTPRequestHandler
import re

# Per-session call counters keyed by AgentRun name (from execution.source).
# Falls back to a global counter for requests without a session key.
session_counters = {}
global_counter = [0]

def get_call_count(req):
    """Return (session_key, current_count) for this request."""
    exec_source = ''
    try:
        exec_source = req.get('execution', {}).get('source', '')
    except (AttributeError, TypeError):
        pass
    # Extract a stable session key (router/react/reflection source label).
    key = exec_source if exec_source else '__global__'
    if key not in session_counters:
        session_counters[key] = 0
    session_counters[key] += 1
    global_counter[0] += 1
    return key, session_counters[key]


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length)

        try:
            req = json.loads(body)
            messages = req.get('messages', [])
            tools = req.get('tools', [])
            system_msg = ''
            for msg in messages:
                if msg.get('role') == 'system':
                    system_msg = msg.get('content', '')
                    break
        except Exception:
            system_msg = ''
            tools = []
            req = {}

        _, call_count = get_call_count(req)

        # Plan-execute: planning phase
        if 'planning phase' in system_msg.lower():
            content = json.dumps([
                {"id": "step-1", "description": "提取现场隐患事实", "action": "分析配电箱和积水情况"},
                {"id": "step-2", "description": "评估风险等级", "action": "根据EHS标准评估电气风险"},
                {"id": "step-3", "description": "生成整改建议", "action": "输出整改工单和安全建议"}
            ])
            message = {
                "role": "assistant",
                "content": content
            }
        # Plan-execute: execution phase
        elif 'execution phase' in system_msg.lower():
            content = json.dumps({
                "status": "completed",
                "result": "配电箱门未关闭，地面有积水，电气风险等级：高",
                "action_taken": "已识别隐患并生成整改建议"
            })
            message = {
                "role": "assistant",
                "content": content
            }
        # Router classifier mode
        elif 'task classifier' in system_msg.lower() or 'classification' in system_msg.lower():
            content = json.dumps({
                "classification": "electrical"
            })
            message = {
                "role": "assistant",
                "content": content
            }
        elif tools and call_count == 1:
            # Function-calling mode: return tool_calls in OpenAI format
            tool_def = tools[0] if tools else {}
            func_name = tool_def.get('function', {}).get('name', 'rectify-ticket-api')
            message = {
                "role": "assistant",
                "content": None,
                "tool_calls": [{
                    "id": "call_mock_001",
                    "type": "function",
                    "function": {
                        "name": func_name,
                        "arguments": json.dumps({
                            "hazard_id": "H-001",
                            "action": "修复配电箱门锁",
                            "priority": "high",
                            "location": "3号线配电间"
                        })
                    }
                }]
            }
            content = None
        elif call_count == 1:
            # ReAct mode: tool call via content JSON
            content = json.dumps({
                "action": "rectify-ticket-api",
                "action_input": {
                    "hazard_id": "H-001",
                    "action": "修复配电箱门锁",
                    "priority": "high",
                    "location": "3号线配电间"
                }
            })
            message = {
                "role": "assistant",
                "content": content
            }
        else:
            # Final answer
            content = json.dumps({
                "final_answer": {
                    "summary": "巡检发现配电箱门未关闭，地面有积水，已创建整改工单",
                    "hazards": [
                        {
                            "title": "配电箱门未关闭",
                            "category": "electrical",
                            "riskLevel": "high",
                            "evidence": ["配电箱门处于打开状态", "地面有积水"],
                            "recommendation": "立即关闭配电箱门并上锁，清理地面积水",
                            "confidence": 0.95
                        }
                    ],
                    "overallRiskLevel": "high",
                    "nextActions": ["通知安全主管", "等待工单完成"],
                    "confidence": 0.95,
                    "needsHumanReview": True
                }
            })
            message = {
                "role": "assistant",
                "content": content
            }

        response = {
            "id": "chatcmpl-mock",
            "object": "chat.completion",
            "choices": [{
                "index": 0,
                "message": message,
                "finish_reason": "stop"
            }]
        }
        resp_body = json.dumps(response).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(resp_body)))
        self.end_headers()
        self.wfile.write(resp_body)

    def do_GET(self):
        self.do_POST()

    def log_message(self, format, *args):
        pass

server = HTTPServer(('0.0.0.0', 8080), Handler)
print('Mock OpenAI server (ReAct + Router + ToolCalling mode) listening on :8080')
server.serve_forever()
