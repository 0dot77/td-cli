def run():
    import json, os
    def _sv(v):
        if v is None: return None
        if isinstance(v,(int,float,bool)): return v
        return str(v)
    types_list = [
        ('TOP', ['noise','constant','composite','moviefilein','text','render','null','switch','select','feedback','glsl','blur','level','ramp','circle','rectangle','cross','edge','flip','hsvadjust','monochrome','over','resolution','transform','crop','cache','in','out','reorder','lookup','limit','math','displace','fit','inside','channelmix','threshold','chromakey','bloom','antialias','layout','tile','videodevicein','cornerpin','screengrab']),
        ('CHOP', ['wave','noise','constant','math','null','select','switch','merge','fan','shuffle','trigger','count','timer','speed','slope','lag','filter','logic','trail','pattern','audiodevicein','audiodeviceout','audiofilein','midiin','analyze','beat','clip','copy','cross','cycle','delay','delete','envelope','expression','extend','function','hold','info','interpolate','keyframe','limit','lookup','mousein','object','parameter','rename','replace','resample','script','shift','sort','spring','stretch','timeslice','trim']),
        ('SOP', ['sphere','box','grid','noise','null','select','merge','transform','circle','line','rectangle','torus','tube','copy','switch','sort','facet','divide','convert','delete','group','hole','clip','extrude','sweep','skin','cap','carve','fit','join','point','polyreduce','project','refine','resample','scatter','script','spring','subdivide','texture','trace','trail','twist','cache','add','boolean','bridge','creep','deform','ends','filein','metaball','peak','primitive','ray','smooth','text']),
        ('DAT', ['text','table','script','web','null','select','switch','merge','chop','sop','top','info','examine','error','fifo','folder','insert','convert','execute','evaluate','opfind','parameter','reorder','substitute','transpose','chopexecute','datexecute','panelexecute']),
        ('COMP', ['base','container','geometry','camera','light','ambient','environment','animation','button','field','slider','list','select','table','op','panel','window','replicator','time']),
        ('MAT', ['phong','pbr','constant','depth','wireframe','pointSprite','line']),
    ]
    parent = op('/project1')
    all_ops = {}
    failed = []
    for family, types in types_list:
        for t in types:
            td_type = t + family
            key = t.lower() + '_' + family.lower()
            try:
                o = parent.create(td_type, '__xtmp')
            except:
                failed.append(td_type)
                continue
            try:
                pars = []
                for p in o.pars():
                    try:
                        pi = {'n':p.name,'l':p.label,'t':type(p.val).__name__,'d':_sv(p.default),'g':p.page.name if p.page else ''}
                        if hasattr(p,'min') and hasattr(p,'max'):
                            pi['min']=_sv(p.min); pi['max']=_sv(p.max)
                        if hasattr(p,'menuNames') and p.menuNames:
                            pi['menu']=list(p.menuNames)
                        pars.append(pi)
                    except: pass
                all_ops[key] = {'name':td_type,'cat':family,'type':o.type,'sum':'','pars':pars}
            except: pass
            finally:
                try: o.destroy()
                except: pass
    out_path = os.path.join(os.path.expanduser('~'),'.td-cli','extracted_operators.json')
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    with open(out_path,'w') as f:
        json.dump(all_ops, f)
    print(json.dumps({'count':len(all_ops),'failed_count':len(failed),'failed':failed}))

run()
