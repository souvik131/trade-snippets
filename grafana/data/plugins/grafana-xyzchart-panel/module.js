define(["@grafana/data","react","@grafana/ui","moment","@emotion/css","react-dom"],((e,t,r,n,l,a)=>(()=>{"use strict";var o,i={154:(e,t,r)=>{r.d(t,{BN:()=>a,Cj:()=>s,N1:()=>o,SO:()=>d,Zc:()=>u,_j:()=>n,c0:()=>l,zT:()=>i});const n=.2,l=2,a=5,o=100,i="YYYY-MM-DD HH:mm:ss",s="#ffffff",u="#808080",d={"super-light-red":"#FFA6B0","light-red":"#FF7383",red:"#F2495C","semi-dark-red":"#E02F44","dark-red":"#C4162A","super-light-orange":"#FFCB7D","light-orange":"#FFB357",orange:"#FF9830","semi-dark-orange":"#FF780A","dark-orange":"#FA6400","super-light-yellow":"#FFF899","light-yellow":"#FFEE52",yellow:"#FADE2A","semi-dark-yellow":"#F2CC0C","dark-yellow":"#E0B400","super-light-green":"#C8F2C2","light-green":"#96D98D",green:"#73BF69","semi-dark-green":"#56A64B","dark-green":"#37872D","super-light-blue":"#C0D8FF","light-blue":"#8AB8FF",blue:"#5794F2","semi-dark-blue":"#3274D9","dark-blue":"#1F60C4","super-light-purple":"#DEB6F2","light-purple":"#CA95E5",purple:"#B877D9","semi-dark-purple":"#A352CC","dark-purple":"#8F3BB8"}},886:(e,t,r)=>{r.d(t,{aA:()=>p,lb:()=>c,oo:()=>m,pw:()=>f,u$:()=>d});var n=r(305),l=r(283),a=r.n(l),o=r(154);function i(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function s(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{},n=Object.keys(r);"function"==typeof Object.getOwnPropertySymbols&&(n=n.concat(Object.getOwnPropertySymbols(r).filter((function(e){return Object.getOwnPropertyDescriptor(r,e).enumerable})))),n.forEach((function(t){i(e,t,r[t])}))}return e}function u(e,t){return t=null!=t?t:{},Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):function(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);r.push.apply(r,n)}return r}(Object(t)).forEach((function(r){Object.defineProperty(e,r,Object.getOwnPropertyDescriptor(t,r))})),e}function d(e,t){if(!e.length)return[];var r,l;const a={frame:null!==(r=null==t?void 0:t.frame)&&void 0!==r?r:0,x:null!==(l=null==t?void 0:t.x)&&void 0!==l?l:null};let o;const i=[];let d=null;for(const t of e[a.frame].fields){const r=(0,n.getFieldDisplayName)(t,e[a.frame],e);if(r!==a.x)if(null===a.x&&[n.FieldType.time,n.FieldType.number].includes(t.type))d=t,a.x=r;else switch(t.type){case n.FieldType.time:i.push(t);break;case n.FieldType.number:o=u(s({},t),{values:new n.ArrayVector(t.values.toArray().map((e=>Number.isFinite(e)||null==e?e:null)))}),i.push(o)}else d=t}return d?[u(s({},e[a.frame]),{fields:[d,...i]})]:[]}function c(e,t){if(!e.length||!t||!t.x&&!t.y&&!t.z)return[];let r,l=null,a=null,o=null;for(const i of e)for(const d of i.fields){const i=(0,n.getFieldDisplayName)(d,e[0],e);let c=null;switch(d.type){case n.FieldType.time:c=d;break;case n.FieldType.number:r=u(s({},d),{values:new n.ArrayVector(d.values.toArray().map((e=>Number.isFinite(e)||null==e?e:null)))}),c=r}c&&(i===t.x&&(l=c),i===t.y&&(a=c),i===t.z&&(o=c))}return l&&a&&o?[u(s({},e[0]),{fields:[l,a,o]})]:[]}function p(e,t){const r=[],l=[];let a={};for(let t of e){if(t.fields.length<3)return{points:new Float32Array,colors:new Float32Array};for(let e=0;e<3;e++){let r=t.fields[e].values.toArray();const n=Math.max(...r),l=Math.min(...r);a[e]={min:l,max:n,factor:(n-l)/o.N1==0?1:(n-l)/o.N1}}}for(let o of e)for(let e=0;e<o.length;e++){for(let t=0;t<3;t++)switch(o.fields[t].type){case n.FieldType.time:case n.FieldType.number:r.push(o.fields[t].values.get(e)/a[t].factor-a[t].min/a[t].factor)}const i=m(t);l.push(i.r),l.push(i.g),l.push(i.b)}return{points:new Float32Array(r),colors:new Float32Array(l)}}function f(e){const t=[],r=[],l=[],i=Math.floor(o.N1/o.BN);if(0===e.length)return{xLabels:t,yLabels:r,zLabels:l};const s=e[0];if(s.fields.length<3)return{xLabels:t,yLabels:r,zLabels:l};const u=s.fields[0].values.toArray(),d=s.fields[1].values.toArray(),c=s.fields[2].values.toArray(),p=Math.min(...u),f=Math.max(...u),m=(f-p)/i,b=Math.min(...d),y=Math.max(...d),g=(y-b)/i,h=Math.min(...c),v=Math.max(...c),F=(v-h)/i;for(let e=0;e<i;e++)s.fields[0].type===n.FieldType.time?t.push(a().unix((p+e*m)/1e3).format(o.zT)):t.push((p+e*m).toFixed(2)),r.push((b+e*g).toFixed(2)),l.push((h+e*F).toFixed(2));return s.fields[0].type===n.FieldType.time?t.push(a().unix(f/1e3).format(o.zT)):t.push(f.toFixed(2)),r.push(y.toFixed(2)),l.push(v.toFixed(2)),{xLabels:t,yLabels:r,zLabels:l}}function m(e){const t=function(e){return"#"===e[0]?e:o.SO[e]}(e);let r=/^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(t);return null===r?{r:1,g:1,b:1}:{r:parseInt(r[1],16)/255,g:parseInt(r[2],16)/255,b:parseInt(r[3],16)/255}}},644:e=>{e.exports=l},305:t=>{t.exports=e},388:e=>{e.exports=r},283:e=>{e.exports=n},650:e=>{e.exports=t},729:e=>{e.exports=a}},s={};function u(e){var t=s[e];if(void 0!==t)return t.exports;var r=s[e]={id:e,loaded:!1,exports:{}};return i[e](r,r.exports,u),r.loaded=!0,r.exports}u.m=i,u.n=e=>{var t=e&&e.__esModule?()=>e.default:()=>e;return u.d(t,{a:t}),t},u.d=(e,t)=>{for(var r in t)u.o(t,r)&&!u.o(e,r)&&Object.defineProperty(e,r,{enumerable:!0,get:t[r]})},u.f={},u.e=e=>Promise.all(Object.keys(u.f).reduce(((t,r)=>(u.f[r](e,t),t)),[])),u.u=e=>e+".js",u.o=(e,t)=>Object.prototype.hasOwnProperty.call(e,t),o={},u.l=(e,t,r,n)=>{if(o[e])o[e].push(t);else{var l,a;if(void 0!==r)for(var i=document.getElementsByTagName("script"),s=0;s<i.length;s++){var d=i[s];if(d.getAttribute("src")==e){l=d;break}}l||(a=!0,(l=document.createElement("script")).charset="utf-8",l.timeout=120,u.nc&&l.setAttribute("nonce",u.nc),l.src=e),o[e]=[t];var c=(t,r)=>{l.onerror=l.onload=null,clearTimeout(p);var n=o[e];if(delete o[e],l.parentNode&&l.parentNode.removeChild(l),n&&n.forEach((e=>e(r))),t)return t(r)},p=setTimeout(c.bind(null,void 0,{type:"timeout",target:l}),12e4);l.onerror=c.bind(null,l.onerror),l.onload=c.bind(null,l.onload),a&&document.head.appendChild(l)}},u.r=e=>{"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})},u.nmd=e=>(e.paths=[],e.children||(e.children=[]),e),u.p="public/plugins/grafana-xyzchart-panel/",(()=>{var e={261:0};u.f.j=(t,r)=>{var n=u.o(e,t)?e[t]:void 0;if(0!==n)if(n)r.push(n[2]);else{var l=new Promise(((r,l)=>n=e[t]=[r,l]));r.push(n[2]=l);var a=u.p+u.u(t),o=new Error;u.l(a,(r=>{if(u.o(e,t)&&(0!==(n=e[t])&&(e[t]=void 0),n)){var l=r&&("load"===r.type?"missing":r.type),a=r&&r.target&&r.target.src;o.message="Loading chunk "+t+" failed.\n("+l+": "+a+")",o.name="ChunkLoadError",o.type=l,o.request=a,n[1](o)}}),"chunk-"+t,t)}};var t=(t,r)=>{var n,l,[a,o,i]=r,s=0;if(a.some((t=>0!==e[t]))){for(n in o)u.o(o,n)&&(u.m[n]=o[n]);i&&i(u)}for(t&&t(r);s<a.length;s++)l=a[s],u.o(e,l)&&e[l]&&e[l][0](),e[l]=0},r=self.webpackChunk=self.webpackChunk||[];r.forEach(t.bind(null,0)),r.push=t.bind(null,r.push.bind(r))})();var d={};return(()=>{u.r(d),u.d(d,{plugin:()=>b});var e=u(305);var t,r=u(650),n=u.n(r),l=u(388),a=u(886);function o(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function i(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{},n=Object.keys(r);"function"==typeof Object.getOwnPropertySymbols&&(n=n.concat(Object.getOwnPropertySymbols(r).filter((function(e){return Object.getOwnPropertyDescriptor(r,e).enumerable})))),n.forEach((function(t){o(e,t,r[t])}))}return e}function s(t){return t.type===e.FieldType.number||t.type===e.FieldType.time}function c(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function p(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{},n=Object.keys(r);"function"==typeof Object.getOwnPropertySymbols&&(n=n.concat(Object.getOwnPropertySymbols(r).filter((function(e){return Object.getOwnPropertyDescriptor(r,e).enumerable})))),n.forEach((function(t){c(e,t,r[t])}))}return e}function f(e,t){return t=null!=t?t:{},Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):function(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);r.push.apply(r,n)}return r}(Object(t)).forEach((function(r){Object.defineProperty(e,r,Object.getOwnPropertyDescriptor(t,r))})),e}!function(e){e[e.NoData=0]="NoData",e[e.BadFrameSelection=1]="BadFrameSelection",e[e.XNotFound=2]="XNotFound"}(t||(t={}));const m=({value:a,onChange:o,context:u})=>{const d=(0,r.useMemo)((()=>{var t;return(null==u||null===(t=u.data)||void 0===t?void 0:t.length)?u.data.map(((t,r)=>({value:r,label:(0,e.getFrameDisplayName)(t,r)}))):[{value:0,label:"First result"}]}),[u.data]);(0,r.useEffect)((()=>{void 0!==(null==a?void 0:a.x)&&o(f(p({},a),{x:void 0}))}),[u.data,null==a?void 0:a.frame]);const c=(0,r.useMemo)((()=>function(e,r){if(!r||!r.length)return{error:t.NoData};var n;e||(e={frame:0});let l=r[null!==(n=e.frame)&&void 0!==n?n:0];if(!l)return{error:t.BadFrameSelection};let a=-1;for(let e=0;e<l.fields.length;e++)if(s(l.fields[e])){a=e;break}const o=l.fields[a],u=[o];for(const e of l.fields)e!==o&&s(e)&&u.push(e);return{frame:(d=i({},l),c={fields:u},c=null!=c?c:{},Object.getOwnPropertyDescriptors?Object.defineProperties(d,Object.getOwnPropertyDescriptors(c)):function(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);r.push.apply(r,n)}return r}(Object(c)).forEach((function(e){Object.defineProperty(d,e,Object.getOwnPropertyDescriptor(c,e))})),d)};var d,c}(a,u.data)),[u.data,a]),m=(0,r.useMemo)((()=>{const t={label:"Not found",value:void 0},r={validFields:[t],xField:t};if(0===u.data.length||!c.frame)return r;let n={validFields:[],xField:{}};var l;const o=u.data?u.data[null!==(l=null==a?void 0:a.frame)&&void 0!==l?l:0]:void 0;if(null==o?void 0:o.fields){const t=o.fields.find((e=>"number"===e.type));if(t){const r=(0,e.getFieldDisplayName)(t,o,u.data);n.xField={label:`${r} (First)`,value:r}}}for(let t of c.frame.fields){const r=(0,e.getFieldDisplayName)(t,o,u.data),l={label:r,value:r};n.validFields.push(l),(null==a?void 0:a.x)&&r===a.x&&(n.xField=l)}return n}),[u.data,c.frame,a]);var b;return n().createElement("div",null,n().createElement(l.Select,{options:d,value:null!==(b=d.find((e=>e.value===(null==a?void 0:a.frame))))&&void 0!==b?b:d[0],onChange:e=>{o(f(p({},a),{frame:e.value}))}}),n().createElement("br",null),n().createElement(l.Label,null,"X Field"),n().createElement(l.Select,{options:m.validFields,value:m.xField,onChange:e=>{o(f(p({},a),{x:e.value}))}}),n().createElement("br",null),n().createElement("br",null))},b=new e.PanelPlugin((e=>{const t=(0,l.useTheme2)(),o=(0,r.useMemo)((()=>"manual"===e.options.seriesMapping?(0,a.lb)(e.data.series,e.options.series):(0,a.u$)(e.data.series,e.options.dims)),[e.data.series,e.options.series,e.options.dims,e.options.seriesMapping]),i=e.options,[s,d]=(0,r.useState)(!1);i.themeColor=t.isDark?"#ffffff":"#000000",i.hudBgColor=t.colors.background.secondary,(0,r.useEffect)((()=>{d(!0)}),[]);let c=!1;for(const e of o)if(e.fields.length<3){c=!0;break}const p=(0,r.lazy)((()=>Promise.all([u.e(290),u.e(723)]).then(u.bind(u,723))));return c||0===o.length?n().createElement("div",{className:"panel-empty"},n().createElement("p",null,"Incorrect data")):n().createElement(n().Fragment,null,s?n().createElement(r.Suspense,{fallback:null},n().createElement(p,{frames:o,options:i})):n().createElement("div",{className:"panel-empty"}))})).setPanelOptions((e=>{e.addRadio({path:"seriesMapping",name:"Series mapping",defaultValue:"auto",settings:{options:[{value:"auto",label:"Auto"},{value:"manual",label:"Manual"}]}}).addCustomEditor({id:"xyPlotConfig",path:"dims",name:"Data",editor:m,showIf:e=>"auto"===e.seriesMapping}).addFieldNamePicker({path:"series.x",name:"X Field",showIf:e=>"manual"===e.seriesMapping}).addFieldNamePicker({path:"series.y",name:"Y Field",showIf:e=>"manual"===e.seriesMapping}).addFieldNamePicker({path:"series.z",name:"Z Field",showIf:e=>"manual"===e.seriesMapping}).addColorPicker({path:"pointColor",name:"Point color",settings:{},defaultValue:"red"}).addNumberInput({path:"pointSize",name:"Point size",settings:{},defaultValue:5})}))})(),d})()));
//# sourceMappingURL=module.js.map