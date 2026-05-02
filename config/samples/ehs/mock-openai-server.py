import json
from http.server import HTTPServer, BaseHTTPRequestHandler

call_count = 0

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        global call_count
        call_count += 1
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length)
        
        try:
            req = json.loads(body)
            messages = req.get('messages', [])
            system_msg = ''
            for msg in messages:
                if msg.get('role') == 'system':
                    system_msg = msg.get('content', '')
                    break
        except:
            system_msg = ''
        
        # Router classifier mode
        if 'task classifier' in system_msg.lower() or 'classification' in system_msg.lower():
            content = json.dumps({
                "classification": "electrical"
            })
        elif call_count == 1:
            # First call: tool call - request rectify-ticket-api
            content = json.dumps({
                "action": "rectify-ticket-api",
                "action_input": {
                    "hazard_id": "H-001",
                    "action": "修复配电箱门锁",
                    "priority": "high",
                    "location": "3号线配电间"
                }
            })
        else:
            # Second call: final answer
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
        
        response = {
            "id": "chatcmpl-mock",
            "object": "chat.completion",
            "choices": [{
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": content
                },
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
print('Mock OpenAI server (ReAct + Router mode) listening on :8080')
server.serve_forever()
