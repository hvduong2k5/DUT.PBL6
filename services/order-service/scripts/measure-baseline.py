"""Small sandbox acceptance/completion baseline, not a production load/HA test."""
import concurrent.futures
import json
import math
import os
from pathlib import Path
import time
import urllib.request
import uuid
import yaml
import jsonschema
ROOT = Path(__file__).resolve().parents[3]
BASE = 'http://127.0.0.1:18004'
MOCK = 'http://127.0.0.1:18104'
INTERNAL = os.getenv('ORDER_INTERNAL_TOKEN', 'order-sandbox-internal-token-2026')
def request(base,path,data=None,token=None,headers=None):
    hdr={'Content-Type':'application/json','X-Internal-Token':INTERNAL}
    if token: hdr['Authorization']='Bearer '+token
    hdr.update(headers or {})
    req=urllib.request.Request(base+path,data=None if data is None else json.dumps(data).encode(),headers=hdr)
    start=time.perf_counter()
    with urllib.request.urlopen(req,timeout=5) as response:
        result=json.load(response)
        return response.status,result,(time.perf_counter()-start)*1000
identity=request(MOCK,'/test/token',{'role':'CUSTOMER'})[1]
token=identity['token']
cart=request(BASE,'/api/v1/cart/items',{'sku_code':'MX-GION-500G','quantity':1,'revision':0},token)[1]
quote=request(BASE,'/api/v1/checkout/quote',{'cart_revision':cart['revision'],'payment_method':'VIETQR','address':{'recipient_name':'Synthetic baseline','phone':'+84905123456','street':'Test only','ward_code':'HUE-01','province_code':'75'}},token)[1]
spec=yaml.safe_load((ROOT/'docs/03_api_specs/order-service.openapi.yaml').read_text(encoding='utf-8'))
schema=spec['components']['schemas']['CheckoutOperation']
schema=json.loads(json.dumps(schema))
for prop in schema['properties'].values():
    if prop.pop('nullable',False):prop['type']=[prop['type'],'null']
validator=jsonschema.Draft7Validator(schema,format_checker=jsonschema.FormatChecker())
def accept(_):
    status,op,elapsed=request(BASE,'/api/v1/checkout',{'quote_id':quote['quote_id'],'cart_revision':cart['revision'],'payment_method':'VIETQR'},token,{'X-Idempotency-Key':str(uuid.uuid4())})
    assert status==202 and op['order_id'] is None
    validator.validate(op)
    return op['operation_id'],elapsed,time.perf_counter()
with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
    results=list(executor.map(accept,range(20)))
completion=[]
orders=[]
for opid,_,accepted in results:
    deadline=time.monotonic()+30
    while time.monotonic()<deadline:
        status,op,_=request(BASE,'/api/v1/checkout-operations/'+opid,token=token)
        validator.validate(op)
        if op['status']=='SUCCEEDED':
            orders.append(op['order_id']);completion.append((time.perf_counter()-accepted)*1000);break
        if op['status'] in ['FAILED','MANUAL_REVIEW']:raise RuntimeError('baseline placement failed: '+op['status'])
        time.sleep(.02)
    else:raise RuntimeError('baseline completion timed out')
# Resolve synthetic unpaid obligations through the same guarded public cancellation API.
for order in orders:
    detail=request(BASE,'/api/v1/orders/'+order,token=token)[1]
    request(BASE,'/api/v1/orders/'+order+'/cancel',{'reason':'baseline fixture cleanup'},token,{'X-Idempotency-Key':str(uuid.uuid4()),'If-Match':str(detail['version'])})
for order in orders:
    deadline=time.monotonic()+30
    while time.monotonic()<deadline:
        state=request(BASE,'/api/v1/orders/'+order,token=token)[1]
        if state['status']=='CANCELLED_BY_USER':break
        time.sleep(.02)
    else:raise RuntimeError('baseline cancellation did not finish')
def pct(values,p):return sorted(values)[max(0,math.ceil(len(values)*p)-1)]
report={'environment':'local Docker sandbox; PostgreSQL/Redis/Kafka real; dependencies simulator','requests':20,'concurrency':5,'scope':'fresh-key acceptance; one-line cart; completion client-observed including polling delay','acceptance_p50_ms':round(pct([r[1] for r in results],.5),2),'acceptance_p95_ms':round(pct([r[1] for r in results],.95),2),'acceptance_p99_ms':round(pct([r[1] for r in results],.99),2),'completion_observed_p95_ms':round(pct(completion,.95),2),'acceptance_target_500ms_met':pct([r[1] for r in results],.95)<=500,'operation_schema_responses_valid':True,'fixture_orders_cancelled':len(orders),'production_slo_proven':False}
(ROOT/'docs/04_testing/order-service/latency-baseline.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
print(json.dumps(report,indent=2))
