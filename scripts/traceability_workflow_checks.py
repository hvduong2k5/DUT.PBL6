"""Real Traceability provider setup and order-flow checks for the isolated MVP stack.

The production Traceability entry point retains HTTPS OIDC and TLS peer checks.
Certificates, credentials, scans and accounts are disposable fixtures only.
"""
import datetime
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import ssl
import subprocess
import time
import urllib.request
import uuid

WAREHOUSE = '00000000-0000-0000-0000-000000000001'
PERMISSIONS = ['traceability.' + p for p in ['read_internal','read_evidence','create','edit','submit','approve','publish','suspend','archive']]


class TraceabilityChecks:
    def __init__(self, root, folder, ports, env, run, start, container, sql, request, wait, suffix):
        self.root, self.folder, self.ports, self.env = root, folder, ports, env
        self.run, self.start, self.container, self.sql, self.request, self.wait = run, start, container, sql, request, wait
        self.suffix = suffix
        self.origin = f"https://127.0.0.1:{ports['trace_public']}"
        self.cert, self.key = folder / 'trace-ca.pem', folder / 'trace-key.pem'
        openssl = shutil.which('openssl')
        if not openssl and os.name == 'nt':
            git = Path(shutil.which('git')).resolve()
            candidate = git.parent.parent / 'usr/bin/openssl.exe'
            if candidate.is_file(): openssl = str(candidate)
        if not openssl: raise RuntimeError('OpenSSL is required for isolated HTTPS acceptance')
        run([openssl,'req','-x509','-newkey','ec','-pkeyopt','ec_paramgen_curve:P-256','-nodes','-days','1',
             '-subj','/CN=localhost','-addext','subjectAltName=DNS:localhost,IP:127.0.0.1',
             '-addext','basicConstraints=critical,CA:TRUE','-addext','keyUsage=critical,digitalSignature,keyCertSign',
             '-keyout',self.key,'-out',self.cert],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        self.cert.chmod(0o644); self.key.chmod(0o600)
        proxy = folder / 'trace-proxy.json'
        proxy.write_text(json.dumps({'cert':str(self.cert),'key':str(self.key),'port':ports['trace_public'],
                                    'keycloak':ports['keycloak'],'web':ports['web'],'storage':ports['storage']}),encoding='utf-8')
        start([shutil.which('node'),root/'scripts/traceability-test-proxy.cjs',proxy],'trace-https')
        self.context = ssl.create_default_context(cafile=str(self.cert))
        self.staff_token, self.source_token, self.resolver_token = [secrets.token_urlsafe(40) for _ in range(3)]
        self.staff_env=dict(IDENTITY_TRACEABILITY_GRPC_ENABLED='true',IDENTITY_TRACEABILITY_GRPC_PORT=str(ports['trace_staff']),
            IDENTITY_TRACEABILITY_GRPC_TLS_CERT_FILE=str(self.cert),IDENTITY_TRACEABILITY_GRPC_TLS_KEY_FILE=str(self.key),
            IDENTITY_TRACEABILITY_GRPC_TOKEN=self.staff_token)
        self.env.update(PEER_TRACEABILITY_TOKEN=self.source_token,
            ORDER_TRACEABILITY_ENABLED='true',ORDER_TRACEABILITY_GRPC=f"127.0.0.1:{ports['trace_grpc']}",
            ORDER_TRACEABILITY_CA_FILE=str(self.cert),ORDER_TRACEABILITY_SERVER_NAME='localhost',
            ORDER_TRACEABILITY_PUBLIC_ORIGIN=self.origin,NEXT_PUBLIC_TRACEABILITY_PUBLIC_ORIGIN=self.origin)
        (folder/'trace-resolver-token').write_text(self.resolver_token,encoding='utf-8')
        self.env['ORDER_TRACEABILITY_TOKEN_FILE']=str(folder/'trace-resolver-token')
        self.env['NODE_EXTRA_CA_CERTS']=str(self.cert)
        self.env['JAVA_TOOL_OPTIONS']='-Xms32m -Xmx384m'
        self.env.update(PEER_ADMIN_ADDRESS=f"127.0.0.1:{ports['trace_inventory']}",
            PEER_ADMIN_ISSUER=self.origin+'/realms/order-auth-test-'+suffix,
            PEER_ADMIN_KEY_FILE=str(folder/'jwt-public.pem'))
        self.env.update(TRACEABILITY_MEDIA_HTTP_UPSTREAM_URL=f"http://127.0.0.1:{ports['trace_media']}/api/v1/media",
            TRACEABILITY_MEDIA_PUBLIC_ORIGIN=self.origin,TRACEABILITY_UPLOAD_CAPABILITY_KEY=secrets.token_urlsafe(40))

    def launch(self, db_name, password, issuer, catalog_token, peer_address, kafka):
        # Separate database and an unprivileged runtime role; no schema-owner API process.
        trace_db='trace_'+self.suffix
        role='trace_runtime_'+self.suffix
        role_secret=secrets.token_urlsafe(32)
        self.sql(f"CREATE DATABASE {trace_db};")
        self.sql(f"CREATE ROLE {role} LOGIN PASSWORD '{role_secret}' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;")
        previous=self.env.copy()
        owner=f"postgres://peer:{password}@127.0.0.1:{self.ports['db']}/{trace_db}?sslmode=disable"
        self.env.update(APP_MODE='integration',DATABASE_URL=owner)
        self.run([self.env['MVP_GO'],'run','./cmd/migrate','-runtime-role',role,'-require-warehouse-writes'],self.root/'services/traceability-service',stdout=subprocess.DEVNULL)
        storage=json.loads((self.folder/'packing-storage-private.json').read_text(encoding='utf-8'))
        media_db='trace_media_'+self.suffix
        self.sql(f'CREATE DATABASE {media_db};')
        media_env=self.folder/'trace-media.env'
        values={'DATABASE_URL':f"postgresql://peer:{password}@host.docker.internal:{self.ports['db']}/{media_db}?schema=public",
            'MINIO_ENDPOINT':'http://'+storage['name']+':9000','MINIO_PUBLIC_ENDPOINT':self.origin,
            'MINIO_ACCESS_KEY':storage['access'],'MINIO_SECRET_KEY':storage['secret'],'MINIO_REGION':'us-east-1','KAFKA_ENABLED':'false'}
        media_env.write_text(''.join(k+'='+v+'\n' for k,v in values.items()),encoding='utf-8'); media_env.chmod(0o600)
        image=self.env.get('TRACEABILITY_TEST_MEDIA_IMAGE','omamx/media-service:traceability-test')
        args=['--network',storage['network'],'--add-host','host.docker.internal:host-gateway','--env-file',str(media_env)]
        self.run(['docker','run','--rm',*args,'--entrypoint','node',image,'node_modules/prisma/build/index.js','migrate','deploy'],stdout=subprocess.DEVNULL)
        self.container('codex-trace-media-'+self.suffix,image,[*args,'-p',f"127.0.0.1:{self.ports['trace_media']}:8007",'-p',f"127.0.0.1:{self.ports['trace_media_grpc']}:9007"])
        self.wait(f"http://127.0.0.1:{self.ports['trace_media']}/fixture-health",expected=404)
        self.env.update(APP_MODE='integration',DATABASE_URL=f"postgres://{role}:{role_secret}@127.0.0.1:{self.ports['db']}/{trace_db}?sslmode=disable",
            HTTP_ADDR=f"127.0.0.1:{self.ports['trace_http']}",PUBLIC_WEB_BASE_URL=self.origin,
            TRACEABILITY_SCOPED_API_ENABLED='true',TRACEABILITY_CURSOR_HMAC_KEY=secrets.token_urlsafe(40),
            TRACEABILITY_ORDER_RESOLVER_ENABLED='true',TRACEABILITY_ORDER_RESOLVER_TOKEN=self.resolver_token,
            GRPC_ADDR=f"127.0.0.1:{self.ports['trace_grpc']}",GRPC_TLS_CERT_FILE=str(self.cert),GRPC_TLS_KEY_FILE=str(self.key),
            TRACEABILITY_OIDC_ISSUER=issuer,TRACEABILITY_OIDC_CA_FILE=str(self.cert),
            TRACEABILITY_IDENTITY_ADDR=f"127.0.0.1:{self.ports['trace_staff']}",TRACEABILITY_IDENTITY_CA_FILE=str(self.cert),TRACEABILITY_IDENTITY_TOKEN=self.staff_token,
            TRACEABILITY_CATALOG_GRPC_ADDR=f"127.0.0.1:{self.ports['catalog_grpc']}",TRACEABILITY_CATALOG_GRPC_PLAINTEXT='true',TRACEABILITY_CATALOG_TOKEN=catalog_token,
            TRACEABILITY_INVENTORY_ADDR=peer_address,TRACEABILITY_INVENTORY_CA_FILE=str(self.folder/'inventory-ca.pem'),TRACEABILITY_INVENTORY_TOKEN=self.source_token,
            TRACEABILITY_MEDIA_ENABLED='true',TRACEABILITY_MEDIA_GRPC_ADDR=f"127.0.0.1:{self.ports['trace_media_grpc']}",TRACEABILITY_MEDIA_GRPC_PLAINTEXT='true',
            OUTBOX_RELAY_ENABLED='true',KAFKA_BROKERS=kafka,EXPIRY_WORKER_ENABLED='true')
        binary=self.folder/('traceability.exe' if os.name=='nt' else 'traceability')
        self.run([self.env['MVP_GO'],'build','-o',binary,'./cmd/server'],self.root/'services/traceability-service')
        self.binary, self.runtime_env = binary, self.env.copy()
        self.process=self.start([binary],'traceability')
        self.env.clear();self.env.update(previous)
        self.env['TRACEABILITY_UPSTREAM_URL']=f"http://127.0.0.1:{self.ports['trace_http']}/api/v1/trace"
        self.wait(f"http://127.0.0.1:{self.ports['trace_http']}/readyz",expected=200)

    def restart(self):
        previous=self.env.copy()
        self.env.clear();self.env.update(self.runtime_env)
        self.process=self.start([self.binary],'traceability')
        self.env.clear();self.env.update(previous)
        self.wait(f"http://127.0.0.1:{self.ports['trace_http']}/readyz")

    def verify_order(self, order_id, staff, customer, accounts, password, login, gateway, web, issuer,
                     profile_url, identity, admin_api, keycloak, admin_password, inventory_schema, purchase):
        staff=login(self.request,gateway,issuer,profile_url,f"http://127.0.0.1:{self.ports['order']}",accounts['E'],password,True,retain=True)
        customer=login(self.request,gateway,issuer,profile_url,f"http://127.0.0.1:{self.ports['order']}",accounts['A'],password,False,retain=True)
        def call(path, body=None, method=None, actor=staff, expected=200):
            status,out,_=self.request(web+path,body,method=method,headers={'Origin':web,'Cookie':'OMAMA_SESSION='+actor['sessionId'],'Idempotency-Key':uuid.uuid4().hex})
            assert status==expected,('Traceability Web operation',path,status,out.get('code') if isinstance(out,dict) else None)
            return out
        def iam(path, body=None, method=None, expected=200):
            status,out,_=self.request(identity+'/api/v1/admin/'+path,body,staff['accessToken'],method,headers={'Idempotency-Key':uuid.uuid4().hex})
            assert status==expected,('IAM operation',path,status)
            return out.get('data')
        role=iam('roles',{'roleCode':'TRACE_E2E_'+self.suffix,'roleName':'Isolated traceability warehouse operator','description':'Acceptance fixture'},'POST',201)
        iam('roles/'+role+'/permissions',{'permissions':PERMISSIONS+['INVENTORY_VIEW','WAREHOUSE_WORKBENCH_VIEW','INVENTORY_RECEIPT_CREATE']},'PUT')
        status,admin,_=self.request(keycloak+'/realms/master/protocol/openid-connect/token',{'grant_type':'password','client_id':'admin-cli','username':'admin','password':admin_password},form=True)
        assert status==200,'Fixture Keycloak admin unavailable'
        def employee(label,warehouse):
            principal=str(uuid.uuid4());email=f'trace-{label}-{self.suffix}@example.test'
            status,_,headers=self.request(admin_api+'/users',{'username':email,'email':email,'enabled':True,'emailVerified':True,
                'firstName':'Trace','lastName':label,'attributes':{'app_user_id':[principal],'principal_type':['EMPLOYEE']},
                'credentials':[{'type':'password','value':password,'temporary':False}]},admin['access_token'],'POST')
            assert status==201,'Fixture employee provisioning failed'
            subject=headers['Location'].rsplit('/',1)[1]
            self.sql(f"INSERT INTO identity.identity_principals(principal_id,idp_issuer,idp_subject,principal_type,provisioning_status) VALUES('{principal}','{issuer}','{subject}','EMPLOYEE','ACTIVE');")
            assignment=iam('role-assignments',{'principalId':principal,'roleId':role,'scopeType':'WAREHOUSE','scopeId':warehouse,'reason':'Isolated acceptance'},'POST')
            account={'principal':principal,'email':email,'subject':subject}
            session=login(self.request,gateway,issuer,profile_url,f"http://127.0.0.1:{self.ports['order']}",account,password,True,retain=True)
            return account,session,assignment
        reviewer_account,reviewer,assignment=employee('reviewer',WAREHOUSE)
        _,foreign,_=employee('other-warehouse','00000000-0000-0000-0000-000000000002')
        pending=call('/api/orders/'+order_id+'/traceability',actor=customer)
        assert pending['state']=='READY' and pending['total']==1 and pending['items'][0]['status']=='NOT_AVAILABLE',('Unpublished batch',pending['state'])
        batch=pending['items'][0]['batch_id']
        call('/api/admin/trace/batches/'+batch+'/context?sku_code=PEER',actor=foreign,expected=403)
        context=call('/api/admin/trace/batches/'+batch+'/context?sku_code=PEER',actor=reviewer)
        # Valid, explicitly synthetic PDF; no external certificate or personal data.
        stream=b'BT /F1 12 Tf 30 100 Td (ISOLATED TEST CERTIFICATE - NO LEGAL VALUE) Tj ET'
        objects=[b'<< /Type /Catalog /Pages 2 0 R >>',b'<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
            b'<< /Type /Page /Parent 2 0 R /MediaBox [0 0 500 150] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>',
            b'<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',b'<< /Length '+str(len(stream)).encode()+b' >>\nstream\n'+stream+b'\nendstream']
        pdf=b'%PDF-1.4\n';offsets=[0]
        for index,obj in enumerate(objects,1):
            offsets.append(len(pdf));pdf+=str(index).encode()+b' 0 obj\n'+obj+b'\nendobj\n'
        xref=len(pdf);pdf+=b'xref\n0 6\n0000000000 65535 f \n'+b''.join(f'{offset:010d} 00000 n \n'.encode() for offset in offsets[1:])
        pdf+=f'trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n'.encode()
        upload=call('/api/admin/trace-media/upload-sessions',{'purpose':'OCOP_CERTIFICATE','fileName':'isolated-certificate.pdf','mediaType':'application/pdf','sizeBytes':len(pdf),'sha256':hashlib.sha256(pdf).hexdigest()},'POST',expected=201)
        with urllib.request.urlopen(urllib.request.Request(upload['uploadUrl'],data=pdf,method='PUT',headers=upload.get('requiredHeaders',{})),context=self.context,timeout=20) as response:
            assert response.status==200,'Object upload failed'
        media_headers={'Origin':web,'Cookie':'OMAMA_SESSION='+staff['sessionId'],'X-Upload-Permit':upload['uploadPermit'],'Idempotency-Key':uuid.uuid4().hex}
        status,_,_=self.request(web+'/api/admin/trace-media/upload-sessions/'+upload['uploadSessionId']+'/complete',{},method='POST',headers=media_headers)
        assert status in (200,202),'Upload completion failed'
        for _ in range(30):
            status,metadata,_=self.request(web+'/api/admin/trace-media/files/'+upload['fileId'],headers=media_headers)
            if status==200 and metadata.get('status')=='READY':break
            time.sleep(.5)
        else:raise AssertionError('Real Media did not reach READY')
        today=datetime.date.today(); evidence=str(uuid.uuid4())
        content={'product_name':'Isolated traceability product','producer':'Test workshop','workshop_location':'Isolated local fixture','artisan_name':None,
            'process':'Acceptance fixture processing','ingredients':[{'name':'Fixture ingredient','origin':'Fixture farm'}],
            'evidence':[{'id':evidence,'reference':'document:'+upload['fileId'],'kind':'CERTIFICATE','description':'Synthetic certificate scan'}],
            'certificates':[{'id':str(uuid.uuid4()),'type':'OCOP','number':'TEST-'+self.suffix,'issuer':'Isolated acceptance issuer','scope':'PRODUCT','scope_id':context['product_id'],
                'issued_on':str(today-datetime.timedelta(days=30)),'valid_until':str(today+datetime.timedelta(days=365)),'no_expiry_stated':False,'review_due_at':None,
                'stars':3,'verification_method':'MANUAL','verified_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'revoked':False,'evidence_ids':[evidence]}],'crafting_video_url':None}
        binding={k:context[k] for k in ['product_id','sku_code','batch_id','source_namespace']}
        profile=call('/api/admin/trace/records',{**binding,'content':content},'POST',expected=201)
        path='/api/admin/trace/records/'+profile['id']
        profile=call(path+'/revisions/1/submit',{'expected_version':profile['version']},'POST')
        call(path+'/revisions/1/approve',{'expected_version':profile['version'],'reason':'Self approval must be denied'},'POST',expected=403)
        profile=call(path+'/revisions/1/approve',{'expected_version':profile['version'],'reason':'Independent fixture review'},'POST',actor=reviewer)
        profile=call(path+'/revisions/1/publish',{'expected_version':profile['version']},'POST',actor=reviewer)
        linked=call('/api/orders/'+order_id+'/traceability',actor=customer)
        assert linked['state']=='READY' and linked['items'][0]['status']=='AVAILABLE' and linked['items'][0]['qr_url']==profile['qr_url'],'Real sealed Order -> resolver link mismatch'
        public=call('/api/customer-core/trace/'+profile['qr_code'],actor=customer)
        assert public['product_name']==content['product_name'],'Published record mismatch'
        original_url=profile['qr_url']
        profile=call(path+'/revisions',{'from_revision':1,'expected_version':profile['version']},'POST')
        changed={**content,'product_name':'UNPUBLISHED CONTENT MUST NOT APPEAR'}
        profile=call(path+'/revisions/2',{'expected_version':profile['version'],'content':changed},'PATCH')
        assert call('/api/customer-core/trace/'+profile['qr_code'],actor=customer)['product_name']==content['product_name'] and profile['qr_url']==original_url,'Draft leaked into public revision'
        work=call('/api/admin/inventory/workbench',actor=reviewer)
        sku=next(row for row in work['skus'] if row['skuCode']=='PEER')
        receipt_key=uuid.uuid4().hex
        receipt_body={'skuId':sku['skuId'],'batchMode':'NEW','batchCode':'TRACE-'+self.suffix,
            'supplierId':str(uuid.uuid4()),'manufacturingDate':str(today-datetime.timedelta(days=1)),
            'expiresAt':str(today+datetime.timedelta(days=20)),'quantity':1,'sourceReference':'TRACE-SOURCE-'+self.suffix,
            'expectedRevision':sku['revision'],'idempotencyKey':receipt_key}
        receipt_headers={'Origin':web,'Cookie':'OMAMA_SESSION='+reviewer['sessionId'],'Idempotency-Key':receipt_key}
        status,receipt,_=self.request(web+'/api/admin/inventory/workbench/receipts',receipt_body,method='POST',headers=receipt_headers)
        assert status==200 and receipt.get('status')=='APPLIED' and receipt.get('warehouseId')==WAREHOUSE,'Scoped receipt binding failed'
        status,replay,_=self.request(web+'/api/admin/inventory/workbench/receipts',receipt_body,method='POST',headers=receipt_headers)
        assert status==200 and replay==receipt,'Receipt replay changed the original result'
        new_batch=receipt['batchId']
        # A second immutable file reference avoids replacing the first profile's Media binding.
        pdf2=pdf+b'\n%additional isolated scan '+self.suffix.encode()
        second_upload=call('/api/admin/trace-media/upload-sessions',{'purpose':'OCOP_CERTIFICATE','fileName':'second-isolated-certificate.pdf','mediaType':'application/pdf',
            'sizeBytes':len(pdf2),'sha256':hashlib.sha256(pdf2).hexdigest()},'POST',actor=reviewer,expected=201)
        with urllib.request.urlopen(urllib.request.Request(second_upload['uploadUrl'],data=pdf2,method='PUT',headers=second_upload.get('requiredHeaders',{})),context=self.context,timeout=20) as response:
            assert response.status==200
        second_headers={'Origin':web,'Cookie':'OMAMA_SESSION='+reviewer['sessionId'],'X-Upload-Permit':second_upload['uploadPermit'],'Idempotency-Key':uuid.uuid4().hex}
        assert self.request(web+'/api/admin/trace-media/upload-sessions/'+second_upload['uploadSessionId']+'/complete',{},method='POST',headers=second_headers)[0] in (200,202)
        for _ in range(30):
            code,meta,_=self.request(web+'/api/admin/trace-media/files/'+second_upload['fileId'],headers=second_headers)
            if code==200 and meta.get('status')=='READY':break
            time.sleep(.5)
        else:raise AssertionError('Second scan did not reach READY')
        context2=call('/api/admin/trace/batches/'+new_batch+'/context?sku_code=PEER',actor=reviewer)
        evidence2=str(uuid.uuid4())
        second_content={**content,'evidence':[{'id':evidence2,'reference':'document:'+second_upload['fileId'],'kind':'CERTIFICATE','description':'Second synthetic scan'}],
            'certificates':[{**content['certificates'][0],'id':str(uuid.uuid4()),'evidence_ids':[evidence2]}]}
        second=call('/api/admin/trace/records',{**{k:context2[k] for k in binding},'content':second_content},'POST',actor=reviewer,expected=201)
        second_path='/api/admin/trace/records/'+second['id']
        second=call(second_path+'/revisions/1/submit',{'expected_version':second['version']},'POST',actor=reviewer)
        second=call(second_path+'/revisions/1/approve',{'expected_version':second['version'],'reason':'Independent second lot review'},'POST')
        second=call(second_path+'/revisions/1/publish',{'expected_version':second['version']},'POST')
        second_order=purchase(customer)
        split=call('/api/orders/'+second_order+'/traceability',actor=customer)
        assert split['state']=='READY' and split['total']==2 and {row['batch_id'] for row in split['items']}=={batch,new_batch},'Real FEFO split did not resolve exact sealed batches'
        assert all(row['packed_quantity']==1 and row['status']=='AVAILABLE' for row in split['items']),'Split quantities or public links are incorrect'
        second=call(second_path+'/suspend',{'expected_version':second['version'],'reason':'Isolated public eligibility test'},'POST')
        split_gone=call('/api/orders/'+second_order+'/traceability',actor=customer)
        assert {row['batch_id']:row['status'] for row in split_gone['items']}=={batch:'AVAILABLE',new_batch:'NO_LONGER_PUBLIC'},'Mixed public/gone batch states changed'
        call('/api/customer-core/trace/'+second['qr_code'],actor=customer,expected=410)
        # Live source changes must not rewrite the immutable sealed packing proof.
        old_code=linked['items'][0]['batch_code']
        self.sql(f'UPDATE "{inventory_schema}".batches SET batch_code=\'LIVE-RENAMED-{self.suffix}\',version=version+1 WHERE id=\'{batch}\';')
        assert call('/api/orders/'+order_id+'/traceability',actor=customer)['items'][0]['batch_code']==old_code,'Order used current batch display data'
        before=call('/api/orders/'+order_id,actor=customer)
        self.process.terminate();self.process.wait(timeout=15)
        unavailable=call('/api/orders/'+order_id+'/traceability',actor=customer)
        assert unavailable['state']=='READY' and unavailable['items'][0]['status']=='UNAVAILABLE','Resolver outage was hidden'
        after=call('/api/orders/'+order_id,actor=customer)
        assert (before['status'],before['version'])==(after['status'],after['version']),'Lookup changed Order lifecycle'
        self.restart()
        assert call('/api/orders/'+order_id+'/traceability',actor=customer)['items'][0]['status']=='AVAILABLE'
        iam('role-assignments/'+assignment+'?expectedVersion=0&reason=Acceptance',method='DELETE')
        status,_,_=self.request(f"http://127.0.0.1:{self.ports['trace_http']}/api/v2/admin/trace/records/{profile['id']}",token=reviewer['accessToken'])
        assert status==403,'Actual IAM revoke did not clamp an unexpired JWT'
        deadline=datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(seconds=60)
        iam('role-assignments',{'principalId':reviewer_account['principal'],'roleId':role,'scopeType':'WAREHOUSE','scopeId':WAREHOUSE,'validUntil':deadline.isoformat(),'reason':'Expiry acceptance'},'POST')
        expiring_session=login(self.request,gateway,issuer,profile_url,f"http://127.0.0.1:{self.ports['order']}",reviewer_account,password,True,retain=True)
        assert self.request(f"http://127.0.0.1:{self.ports['trace_http']}/api/v2/admin/trace/records/{profile['id']}",token=expiring_session['accessToken'])[0]==200,'Valid scoped assignment was not usable'
        while datetime.datetime.now(datetime.timezone.utc)<=deadline:time.sleep(.5)
        assert self.request(f"http://127.0.0.1:{self.ports['trace_http']}/api/v2/admin/trace/records/{profile['id']}",token=expiring_session['accessToken'])[0]==403,'Actual IAM expiry did not clamp an unexpired JWT'
        private={'web':web,'public_origin':self.origin,'order_id':order_id,'profile_id':profile['id'],'qr_url':original_url,
            'second_order_id':second_order,'gone_qr_url':second['qr_url'],'customer_session':customer['sessionId'],'staff_session':staff['sessionId'],'password':password,'accounts':accounts}
        (self.folder/'trace-browser-private.json').write_text(json.dumps(private),encoding='utf-8');(self.folder/'trace-browser-private.json').chmod(0o600)
        self.result={'status':'PASS','order_id':order_id,'profile_id':profile['id'],'sealed_batch_id':batch,'qr_url':original_url,
            'second_order_id':second_order,'second_profile_id':second['id'],'real_receipt_replay':True,'real_two_batch_fefo':True,'mixed_public_gone':True,'iam_api_expiry':True,'real_certificate_upload':True,'independent_review':True,'warehouse_denial':True,'iam_api_revoke':True,
            'draft_public_isolation':True,'immutable_sealed_display':True,'resolver_outage_keeps_order_detail':True}
        (self.folder/'traceability-result.json').write_text(json.dumps(self.result,indent=2),encoding='utf-8')
        print('PASS: actual Media upload, warehouse context, independent review/publication, sealed Order link, draft isolation, resolver outage and IAM API revoke',flush=True)
