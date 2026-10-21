<template>
  <div class="legal-container">
    <div class="legal-card">
      <h1>隐私政策</h1>
      <p class="update-time">更新日期：2025-11-16</p>

      <section>
        <h2>一、引言</h2>
        <p>
          本隐私政策旨在向您说明我们如何收集、使用、共享和保护您的个人信息。我们深知个人信息对您的重要性，并承诺按照合法、正当、必要、诚信的原则处理您的信息。</p>
      </section>

      <section>
        <h2>二、我们收集的信息</h2>
        <ol>
          <li>注册信息：包括用户名、邮箱地址、登录密码等。</li>
          <li>学生信息（可选）：如学号、用于外部系统登录的密码（加密存储）。</li>
          <li>使用数据：您在系统中的作业、测试、代办等任务数据，及操作日志（用于改善服务与故障排查）。</li>
          <li>设备与网络信息：包括设备类型、浏览器类型、IP 地址、请求时间等，用于保障系统安全与统计分析。</li>
        </ol>
      </section>

      <section>
        <h2>三、信息的使用目的</h2>
        <ol>
          <li>提供与维护服务：包括账户管理、任务记录与提醒、数据同步。</li>
          <li>优化用户体验与安全：通过日志与统计数据进行性能优化、异常检测与安全审计。</li>
          <li>沟通与通知：向您发送与账户、服务变更、安全提醒相关的必要通知。</li>
        </ol>
      </section>

      <section>
        <h2>四、信息的共享与披露</h2>
        <p>我们不会向无关第三方出售、出租、交易您的个人信息。仅在以下情形下可能共享或披露：</p>
        <ol>
          <li>获得您的明确同意后；</li>
          <li>基于法律法规、行政或司法机关要求；</li>
          <li>在必要范围内与服务提供商合作（如邮件发送服务），我们会与其签署保密协议，并要求其仅为实现目的而使用相关信息；
          </li>
          <li>为保护您、我们或公众的合法权益、财产或安全所必需。</li>
        </ol>
      </section>

      <section>
        <h2>五、信息的存储与安全</h2>
        <ol>
          <li>敏感信息（如外部系统密码）将采用加密方式存储，避免明文保存与传输。</li>
          <li>我们采取访问控制、日志审计、网络隔离等安全措施降低信息泄露、滥用、变更与丢失的风险。</li>
          <li>尽管我们已采取合理的安全保障，但任何系统均无法保证 100% 安全。</li>
        </ol>
      </section>

      <section>
        <h2>六、您的权利</h2>
        <ol>
          <li>访问与更正：您可以在设置页面查看与更新您的账户信息（如邮箱、学号、外部系统密码）。</li>
          <li>删除与撤���：在符合法律法规的前提下，您可以请求删除部分信息或撤回授权。</li>
          <li>注销账户：您可以联系我们申请注销账户，注销后我们将按照法律法规要求处理您的信息。</li>
        </ol>
      </section>

      <section>
        <h2>七、未成年人保护</h2>
        <p>如您为未成年人，请在监护人指导下使用本服务。我们将根据法律法规加强未成年人个人信息的保护。</p>
      </section>

      <section>
        <h2>八、政策更新</h2>
        <p>
          我们可能适时对本隐私政策进行修订。更新后的政策将通过页面公告或其他合理方式进行提示，修订在公告载明的日期生效。</p>
      </section>

      <section>
        <h2>九、联系我们</h2>
        <p>如您对本隐私政策或个人信息保护相关事宜有任何疑问、意见或建议，请通过设置页提供的联系方式与我们取得联系。</p>
      </section>

      <section>
        <h2>十、联系方式</h2>
        <p>针对本隐私政策或个人信息保护事宜，欢迎通过以下方式反馈：</p>
        <ul>
          <li>电子邮箱：<a href="mailto:lingxin@lingxin.org">lingxin@lingxin.org</a></li>
          <li>GitHub：<a href="https://github.com/ling0727-ai" rel="noopener" target="_blank">ling0727-ai</a></li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script>
export default {
  name: 'PrivacyPolicy'
}
</script>

<style scoped>
.legal-container {
  min-height: 100vh;
  background: #f5f7fb;
  padding: 24px 16px;
}

.legal-card {
  max-width: 900px;
  margin: 0 auto;
  background: #fff;
  border-radius: 16px;
  padding: 28px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.06);
}

h1 {
  margin: 0 0 8px;
  font-size: 28px;
  color: #111827;
}

.update-time {
  margin: 0 0 20px;
  color: #6b7280;
  font-size: 14px;
}

section {
  margin: 18px 0;
}

h2 {
  font-size: 18px;
  margin: 0 0 8px;
  color: #0f172a;
}

p, li {
  color: #374151;
  line-height: 1.8;
}

ol {
  padding-left: 18px;
}
</style>

