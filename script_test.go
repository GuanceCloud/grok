package grok

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

var _ = script1

var script1 = [][3]string{

	// #pattern
	{`add_pattern("GREEDYLINES","(?s).*")`,
		"GREEDYLINES",
		"(?s).*"},
	{`add_pattern("GREEDYLINE","(?:(.|\r|\n|(?m))*)")`,
		"GREEDYLINE",
		`(?:(.|\r|\n|(?m))*)`},
	// #INFO AND WARN
	{`grok(_,"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINES:msg}")`,
		"_",
		"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINES:msg}"},

	{`grok(_,"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class} - %{DATA:service_name} %{DATA:trace_id} %{DATA:span_id} -  -%{SPACE}%{GREEDYLINE:msg}")`,
		"_",
		"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class} - %{DATA:service_name} %{DATA:trace_id} %{DATA:span_id} -  -%{SPACE}%{GREEDYLINE:msg}"},

	// #二维取消率
	{`add_pattern("state_name","(flight_order_status_metric)|(flight_validate_status_metric)|(hotel_validate_status_metric)|(train_order_status_metric)|(train_channel_query_metric)|(train_channel_order_metric)|(train_change_status_metric)|(train_refund_apply_metric)|(train_account_login_metric)|(train_get_contact_metric)|(hcar_supplier_createOrder_metric)|(hcar_supplier_cancelOrder_metric)")`,
		"state_name",
		"(flight_order_status_metric)|(flight_validate_status_metric)|(hotel_validate_status_metric)|(train_order_status_metric)|(train_channel_query_metric)|(train_channel_order_metric)|(train_change_status_metric)|(train_refund_apply_metric)|(train_account_login_metric)|(train_get_contact_metric)|(hcar_supplier_createOrder_metric)|(hcar_supplier_cancelOrder_metric)"},

	{`grok(msg,"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value},%{NOTSPACE:supplier}")`,
		"msg",
		"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value},%{NOTSPACE:supplier}"},

	// #一维取消率
	{`add_pattern("state_name","(flight_change_nature_metric)|(flight_refund_nature_metric)|(flight_search_book_metric)|(flight_book_change_metric)|(flight_create_approve_metric)|(train_approve_submit_metric)|(train_discount_use_metric)|(hotel_detail_metric)|(hotel_order_create_metric)|(hotel_search_metric)|(hotel_order_refuse_metric)|(hotel_order_cancel_metric)|(hcar_user_estimate_metric)|(hcar_user_createOrder_metric)|(hcar_user_cancelOrder_metric)|(hcar_supplier_takeOrder_metric)|(hcar_orderCancel_supplier_metric)")`,
		"state_name",
		"(flight_change_nature_metric)|(flight_refund_nature_metric)|(flight_search_book_metric)|(flight_book_change_metric)|(flight_create_approve_metric)|(train_approve_submit_metric)|(train_discount_use_metric)|(hotel_detail_metric)|(hotel_order_create_metric)|(hotel_search_metric)|(hotel_order_refuse_metric)|(hotel_order_cancel_metric)|(hcar_user_estimate_metric)|(hcar_user_createOrder_metric)|(hcar_user_cancelOrder_metric)|(hcar_supplier_takeOrder_metric)|(hcar_orderCancel_supplier_metric)"},

	{`grok(msg,"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value}")`,
		"msg",
		"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value}"},

	// #数字类型指标统计
	{`add_pattern("state_name","(flight_abe_search_metric)|(train_order_login_time_difference_metric)")`,
		"state_name",
		"(flight_abe_search_metric)|(train_order_login_time_difference_metric)"},

	{`grok(msg,"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_int_value}")`,
		`msg`,
		"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_int_value}"},

	// # #ERROR Stack
	{`grok(_,"%{TIMESTAMP_ISO8601:time}\\s{1}%{LOGLEVEL:status}\\s{1}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINE:msg}(\\n)(%{GREEDYLINES:stack_trace})")`,
		`_`,
		"%{TIMESTAMP_ISO8601:time}\\s{1}%{LOGLEVEL:status}\\s{1}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINE:msg}(\\n)(%{GREEDYLINES:stack_trace})"},
}

var script = [][3]string{

	// #pattern
	{`add_pattern("GREEDYLINES","(?s).*")`,
		"GREEDYLINES",
		"(?s).*"},
	{`add_pattern("GREEDYLINE","(?:(.|\r|\n|(?m))*)")`,
		"GREEDYLINE",
		`(?:(.|\r|\n|(?m))*)`},
	// #INFO AND WARN
	{`grok(_,"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINES:msg}")`,
		"_",
		"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINES:msg}"},

	{`grok(_,"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class} - %{DATA:service_name} %{DATA:trace_id} %{DATA:span_id} -  -%{SPACE}%{GREEDYLINE:msg}")`,
		"_",
		"%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}%{NOTSPACE:class} - %{DATA:service_name} %{DATA:trace_id} %{DATA:span_id} -  -%{SPACE}%{GREEDYLINE:msg}"},

	// #二维取消率
	{`add_pattern("state_name","(flight_order_status_metric)|(flight_validate_status_metric)|(hotel_validate_status_metric)|(train_order_status_metric)|(train_channel_query_metric)|(train_channel_order_metric)|(train_change_status_metric)|(train_refund_apply_metric)|(train_account_login_metric)|(train_get_contact_metric)|(hcar_supplier_createOrder_metric)|(hcar_supplier_cancelOrder_metric)")`,
		"state_name",
		"(?:flight_order_status_metric)|(?:flight_validate_status_metric)|(?:hotel_validate_status_metric)|(?:train_order_status_metric)|(?:train_channel_query_metric)|(?:train_channel_order_metric)|(?:train_change_status_metric)|(?:train_refund_apply_metric)|(?:train_account_login_metric)|(?:train_get_contact_metric)|(?:hcar_supplier_createOrder_metric)|(?:hcar_supplier_cancelOrder_metric)"},

	{`grok(msg,"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value},%{NOTSPACE:supplier}")`,
		"msg",
		"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value},%{NOTSPACE:supplier}"},

	// #一维取消率
	{`add_pattern("state_name","(?:flight_change_nature_metric)|(?:flight_refund_nature_metric)|(?:flight_search_book_metric)|(?:flight_book_change_metric)|(?:flight_create_approve_metric)|(?:train_approve_submit_metric)|(?:train_discount_use_metric)|(?:hotel_detail_metric)|(?:hotel_order_create_metric)|(?:hotel_search_metric)|(?:hotel_order_refuse_metric)|(?:hotel_order_cancel_metric)|(?:hcar_user_estimate_metric)|(?:hcar_user_createOrder_metric)|(?:hcar_user_cancelOrder_metric)|(?:hcar_supplier_takeOrder_metric)|(?:hcar_orderCancel_supplier_metric)")`,
		"state_name",
		"(?:flight_change_nature_metric)|(?:flight_refund_nature_metric)|(?:flight_search_book_metric)|(?:flight_book_change_metric)|(?:flight_create_approve_metric)|(?:train_approve_submit_metric)|(?:train_discount_use_metric)|(?:hotel_detail_metric)|(?:hotel_order_create_metric)|(?:hotel_search_metric)|(?:hotel_order_refuse_metric)|(?:hotel_order_cancel_metric)|(?:hcar_user_estimate_metric)|(?:hcar_user_createOrder_metric)|(?:hcar_user_cancelOrder_metric)|(?:hcar_supplier_takeOrder_metric)|(?:hcar_orderCancel_supplier_metric)"},

	{`grok(msg,"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value}")`,
		"msg",
		"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_value}"},

	// #数字类型指标统计
	{`add_pattern("state_name","(flight_abe_search_metric)|(train_order_login_time_difference_metric)")`,
		"state_name",
		"(?:flight_abe_search_metric)|(?:train_order_login_time_difference_metric)"},

	{`grok(msg,"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_int_value}")`,
		`msg`,
		"%{state_name:state_name}%{SPACE}:%{NOTSPACE:state_int_value}"},

	// # #ERROR Stack
	{`grok(_,"^%{TIMESTAMP_ISO8601:time}\\s{1}%{LOGLEVEL:status}\\s{1}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINE:msg}(\\n)(%{GREEDYLINES:stack_trace})")`,
		`_`,
		"%{TIMESTAMP_ISO8601:time}\\s{1}%{LOGLEVEL:status}\\s{1}%{NOTSPACE:class}%{SPACE}-%{SPACE}%{GREEDYLINE:msg}(\\n)(%{GREEDYLINES:stack_trace})"},
}

var logdata = `2024-06-17 18:26:59,982 INFO  com.hose.mall.business.api.open.notice.csc.RealmCallback - mall-business-api 578121519100746035 5663413048180047915 -  - 查询企业（）开通产品:{"success":true,"errorMessage":"","data":"[{\"id\":\"meituan\",\"name\":\"外卖\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-01-17/16739407390136787.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/wmmeal/address-list\",\"busType\":0},{\"id\":\"flight\",\"name\":\"机票\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-01-17/16739407652420321.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/wflight\",\"busType\":0},{\"id\":\"canyin\",\"name\":\"餐饮\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-02-24/16772074757537697.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/wmmeal/redirect-mt\",\"busType\":0},{\"id\":\"hcar\",\"name\":\"用车\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-01-17/16739407571493419.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/wmhcar\",\"busType\":0},{\"id\":\"ibs\",\"name\":\"国际业务\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-06-29/16880060050058542.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/wmactivity/ibs\",\"busType\":0},{\"id\":\"train\",\"name\":\"火车票\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-01-17/16739407286374845.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/wtrain\",\"busType\":0},{\"id\":\"hotel\",\"name\":\"酒店\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-01-17/16739407490309778.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/whotel\",\"busType\":0},{\"id\":\"wmshop\",\"name\":\"企业购\",\"companyId\":\"1178655197065510912\",\"iconUrl\":\"https://hose-mall-image.oss-cn-beijing.aliyuncs.com/module_operation/business_flight_ticket_refund/2023-02-23/16771355827759774.png\",\"actionUrl\":\"https://mall-app.ekuaibao.com/wmshop\",\"busType\":0}]"} [TID: N/A]`

func BenchmarkScript(b *testing.B) {
	var gpLi []struct {
		name string
		re   *GrokRegexp
	}

	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Fatal(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}
	for _, v := range script {
		if strings.HasPrefix(v[0], "add_pattern") {
			p, err := DenormalizePattern(v[2], PatternStorage{de})
			if err != nil {
				b.Fatal(err)
			}
			de[v[1]] = p
		} else {
			gp, err := CompilePattern(v[2], PatternStorage{de})
			if err != nil {
				b.Fatal(err)
			}
			gpLi = append(gpLi, struct {
				name string
				re   *GrokRegexp
			}{
				name: v[1],
				re:   gp,
			})
		}
	}

	b.ResetTimer()

	fn := func() {
		message := logdata
		var msg string

		for _, v := range gpLi {

			var mp []string
			var err error
			switch v.name {
			case "_":
				mp, err = v.re.Run(message, true)
				if errors.Is(err, ErrNotCompiled) {
					b.Fatal(err)
				}
			case "msg":
				mp, err = v.re.Run(msg, true)
				if errors.Is(err, ErrNotCompiled) {
					b.Fatal(err)
				}
			}

			if val, ok := v.re.GetValCastByName("msg", mp); ok {
				if val, ok := val.(string); ok {
					msg = val
				}
			}
		}
	}

	b.Run("script-std", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			fn()
		}
	})

	b.Log(len(gpLi))
}

func TestScript(t *testing.T) {
	de, errs := DenormalizePatternsFromMap(CopyDefalutPatterns())
	if len(errs) != 0 {
		fmt.Print(errs)
		return
	}
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{de})
	if err != nil {
		fmt.Print(err)
	}
	ret, err := g.Run(`127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`, true)
	if err != nil {
		fmt.Print(err)
	}
	for k, name := range g.MatchNames() {
		fmt.Printf("%+15s: %s\n", name, ret[k])
	}
}
